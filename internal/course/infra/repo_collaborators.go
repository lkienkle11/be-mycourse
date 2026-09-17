package infra

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"

	authzdomain "mycourse-io-be/internal/authorization/domain"
	courseapp "mycourse-io-be/internal/course/application"
	"mycourse-io-be/internal/course/domain"
	instructordomain "mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/userpicker"
	"mycourse-io-be/internal/shared/utils"
)

// collaboratorSourceRow is one collaborator: the synthesized owner (never a stored binding,
// matching CoursePolicyProvider's decision) or an active EDITOR role binding. SortAt (each
// side's created_at) preserves the original "owner first, then insertion order" display
// ordering that course_collaborators.id used to provide.
type collaboratorSourceRow struct {
	UserID string
	Role   string
	SortAt int64
}

// collaboratorSourceRows reads bindings through RoleBindingService.List, the one sanctioned
// read path for authorization_role_bindings — never a direct query against that table.
func (r *GormRepository) collaboratorSourceRows(ctx context.Context, db *gorm.DB, courseID string) ([]collaboratorSourceRow, error) {
	var course struct {
		OwnerUserID string `gorm:"column:owner_user_id"`
		CreatedAt   int64  `gorm:"column:created_at"`
	}
	if err := db.WithContext(ctx).Table(constants.TableCourses).
		Select("owner_user_id, created_at").Where("id = ?", courseID).Take(&course).Error; err != nil {
		return nil, err
	}
	bindings, err := r.roleBindings.List(gormx.WithTx(ctx, db), authzdomain.RoleBindingQuery{
		Resource: authzdomain.Resource{Type: courseapp.ResourceTypeCourse, ID: courseID},
	})
	if err != nil {
		return nil, err
	}
	out := make([]collaboratorSourceRow, 0, len(bindings)+1)
	out = append(out, collaboratorSourceRow{UserID: course.OwnerUserID, Role: domain.CollaboratorRoleOwner, SortAt: course.CreatedAt})
	for _, b := range bindings {
		out = append(out, collaboratorSourceRow{UserID: b.PrincipalUserID, Role: b.RoleName, SortAt: b.CreatedAt})
	}
	sortCollaboratorSources(out)
	return out, nil
}

// sortCollaboratorSources orders rows the way collaborators have always displayed: the
// canonical owner first, then by original grant/creation order.
func sortCollaboratorSources(rows []collaboratorSourceRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		iOwner := rows[i].Role == domain.CollaboratorRoleOwner
		jOwner := rows[j].Role == domain.CollaboratorRoleOwner
		if iOwner != jOwner {
			return iOwner
		}
		return rows[i].SortAt < rows[j].SortAt
	})
}

type collaboratorProfileScanRow struct {
	UserID       string `gorm:"column:user_id"`
	DisplayName  string `gorm:"column:display_name"`
	Email        string `gorm:"column:email"`
	AvatarFileID string `gorm:"column:avatar_file_id"`
	AvatarURL    string `gorm:"column:avatar_url"`
}

const collaboratorProfileSelectSQL = `
SELECT
    u.id::text AS user_id,
    COALESCE(u.display_name, '') AS display_name,
    COALESCE(u.email, '') AS email,
    COALESCE(u.avatar_file_id::text, '') AS avatar_file_id,
    COALESCE(m.url, '') AS avatar_url
FROM users u
LEFT JOIN media_files m ON m.id = u.avatar_file_id AND m.deleted_at IS NULL
WHERE u.deleted_at IS NULL AND u.id IN @user_ids`

// loadCollaboratorProfiles resolves display profiles for sources (already ordered
// owner-first/by-grant-time), applying the active-user filter and/or search when requested, and
// preserves sources' order in its output (rows for a user profile that no longer exists, e.g.
// deleted, are simply dropped — matching the prior INNER JOIN users' behavior).
func (r *GormRepository) loadCollaboratorProfiles(
	ctx context.Context,
	db *gorm.DB,
	sources []collaboratorSourceRow,
	applyActiveFilter bool,
	search string,
) ([]domain.Collaborator, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	roleByUserID := make(map[string]string, len(sources))
	userIDs := make([]string, len(sources))
	for i, s := range sources {
		roleByUserID[s.UserID] = s.Role
		userIDs[i] = s.UserID
	}
	q := collaboratorProfileSelectSQL
	args := map[string]any{"user_ids": userIDs}
	if applyActiveFilter {
		q += userpicker.ActiveUserWhereClause()
		args["now"] = timex.NowUnix()
	}
	searchClause, searchArgs := utils.UserDisplayNameEmailSearchSQL(search)
	q += searchClause
	for k, v := range searchArgs {
		args[k] = v
	}
	var rows []collaboratorProfileScanRow
	if err := db.WithContext(ctx).Raw(q, args).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byUserID := make(map[string]collaboratorProfileScanRow, len(rows))
	for _, row := range rows {
		byUserID[row.UserID] = row
	}
	out := make([]domain.Collaborator, 0, len(rows))
	for _, s := range sources {
		row, ok := byUserID[s.UserID]
		if !ok {
			continue
		}
		out = append(out, domain.Collaborator{
			UserID: row.UserID, Role: roleByUserID[row.UserID], DisplayName: row.DisplayName,
			Email: row.Email, AvatarFileID: row.AvatarFileID, AvatarURL: row.AvatarURL,
		})
	}
	return out, nil
}

func (r *GormRepository) loadCollaboratorsByUserIDs(ctx context.Context, db *gorm.DB, courseID string, userIDs []string) ([]domain.Collaborator, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	sources, err := r.collaboratorSourceRows(ctx, db, courseID)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		wanted[id] = struct{}{}
	}
	filtered := make([]collaboratorSourceRow, 0, len(userIDs))
	for _, s := range sources {
		if _, ok := wanted[s.UserID]; ok {
			filtered = append(filtered, s)
		}
	}
	return r.loadCollaboratorProfiles(ctx, db, filtered, false, "")
}

// paginateCollaborators slices an already-ordered, already-filtered collaborator list for one
// page, matching utils.ParsedListFilter's Offset/PerPage semantics.
func paginateCollaborators(collaborators []domain.Collaborator, offset, perPage int) []domain.Collaborator {
	if offset > len(collaborators) {
		offset = len(collaborators)
	}
	end := offset + perPage
	if end > len(collaborators) {
		end = len(collaborators)
	}
	return collaborators[offset:end]
}

func (r *GormRepository) ListCollaborators(ctx context.Context, courseID string, actorUserID string, filter domain.CollaboratorListFilter) ([]domain.Collaborator, int64, error) {
	db := r.db.WithContext(ctx)
	if _, err := r.requireEditorAccess(ctx, db, courseID, actorUserID, courseapp.ActionCourseCollaboratorsView); err != nil {
		return nil, 0, err
	}
	sources, err := r.collaboratorSourceRows(ctx, db, courseID)
	if err != nil {
		return nil, 0, err
	}
	matched, err := r.loadCollaboratorProfiles(ctx, db, sources, true, filter.Search)
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(matched))
	parsed := utils.ParseListFilter(utils.BaseFilter{Page: filter.Page, PerPage: filter.PerPage})
	return paginateCollaborators(matched, parsed.Offset, parsed.PerPage), total, nil
}

// instructorCandidatesBaseSQL excludes the course owner and every active EDITOR role-binding
// principal for this course from the instructor picker (the role-gate equivalent of the old
// course_collaborators exclusion subquery). The caller supplies the excluded id set as
// @excluded_ids (via RoleBindingService.List, never a direct query against
// authorization_role_bindings from here).
func instructorCandidatesBaseSQL() string {
	return userpicker.UserPickerSelectSQL(constants.TableAppUsers) + fmt.Sprintf(`
INNER JOIN %s ur ON ur.user_id = u.id
INNER JOIN %s ro ON ro.id = ur.role_id AND ro.name = @role_name
WHERE u.deleted_at IS NULL%s
  AND u.id NOT IN @excluded_ids`, constants.TableRBACUserRoles, constants.TableRBACRoles, userpicker.EligiblePickerWhereClause())
}

func (r *GormRepository) ListInstructorCandidates(ctx context.Context, courseID string, actorUserID string, filter domain.InstructorCandidateFilter) ([]domain.InstructorCandidate, int64, error) {
	var ownerUserID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID, courseapp.ActionCourseCollaboratorsManage)
		if err != nil {
			return err
		}
		ownerUserID = access.OwnerUserID
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	bindings, err := r.roleBindings.List(ctx, authzdomain.RoleBindingQuery{
		Resource: authzdomain.Resource{Type: courseapp.ResourceTypeCourse, ID: courseID},
	})
	if err != nil {
		return nil, 0, err
	}
	excludedIDs := make([]string, 0, len(bindings)+1)
	excludedIDs = append(excludedIDs, ownerUserID)
	for _, b := range bindings {
		excludedIDs = append(excludedIDs, b.PrincipalUserID)
	}
	rows, total, err := userpicker.ListRows(ctx, r.db, instructorCandidatesBaseSQL(), map[string]any{
		"excluded_ids": excludedIDs,
		"role_name":    instructordomain.RoleNameInstructor,
		"now":          timex.NowUnix(),
	}, userpicker.ListFilter{Page: filter.Page, PerPage: filter.PerPage, Search: filter.Search})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.InstructorCandidate, len(rows))
	for i, row := range rows {
		out[i] = domain.InstructorCandidate{
			UserID: row.UserID, DisplayName: row.DisplayName, Email: row.Email,
			AvatarFileID: row.AvatarFileID, AvatarURL: row.AvatarURL,
		}
	}
	return out, total, nil
}
