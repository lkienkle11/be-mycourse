package infra

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	instructordomain "mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/userpicker"
	"mycourse-io-be/internal/shared/utils"
)

type collaboratorScanRow struct {
	UserID       string `gorm:"column:user_id"`
	Role         string `gorm:"column:role"`
	DisplayName  string `gorm:"column:display_name"`
	Email        string `gorm:"column:email"`
	AvatarFileID string `gorm:"column:avatar_file_id"`
	AvatarURL    string `gorm:"column:avatar_url"`
}

const collaboratorsSelectSQL = `
SELECT
    cc.user_id,
    cc.role,
    COALESCE(u.display_name, '') AS display_name,
    COALESCE(u.email, '') AS email,
    COALESCE(u.avatar_file_id::text, '') AS avatar_file_id,
    COALESCE(m.url, '') AS avatar_url
FROM course_collaborators cc
INNER JOIN users u
    ON u.id = cc.user_id AND u.deleted_at IS NULL
LEFT JOIN media_files m
    ON m.id = u.avatar_file_id AND m.deleted_at IS NULL
WHERE cc.course_id = @course_id AND cc.deleted_at IS NULL`

func collaboratorOrderSQL() string {
	return " ORDER BY CASE WHEN cc.role = 'OWNER' THEN 0 ELSE 1 END, cc.id ASC"
}

func (r *GormRepository) loadCollaboratorsByUserIDs(ctx context.Context, db *gorm.DB, courseID string, userIDs []string) ([]domain.Collaborator, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	baseSQL := strings.Replace(collaboratorsSelectSQL, "@course_id", "?", 1)
	q := baseSQL + " AND cc.user_id IN ?" + collaboratorOrderSQL()
	var rows []collaboratorScanRow
	if err := db.WithContext(ctx).Raw(q, courseID, userIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return scanRowsToCollaborators(rows), nil
}

func scanRowsToCollaborators(rows []collaboratorScanRow) []domain.Collaborator {
	out := make([]domain.Collaborator, len(rows))
	for i, row := range rows {
		out[i] = domain.Collaborator{
			UserID: row.UserID, Role: row.Role, DisplayName: row.DisplayName, Email: row.Email,
			AvatarFileID: row.AvatarFileID, AvatarURL: row.AvatarURL,
		}
	}
	return out
}

func (r *GormRepository) ListCollaborators(ctx context.Context, courseID string, actorUserID string, filter domain.CollaboratorListFilter) ([]domain.Collaborator, int64, error) {
	db := r.db.WithContext(ctx)
	if _, err := r.requireCourseAccess(ctx, db, courseID, actorUserID); err != nil {
		return nil, 0, err
	}
	parsed := utils.ParseListFilter(utils.BaseFilter{
		Page:    filter.Page,
		PerPage: filter.PerPage,
	})
	searchClause, searchArgs := utils.UserDisplayNameEmailSearchSQL(filter.Search)
	countQ := collaboratorsSelectSQL + searchClause
	args := map[string]any{"course_id": courseID}
	for k, v := range searchArgs {
		args[k] = v
	}
	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM ("+countQ+") AS collaborators", args).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	listQ := countQ + collaboratorOrderSQL() + fmt.Sprintf(" LIMIT %d OFFSET %d", parsed.PerPage, parsed.Offset)
	var rows []collaboratorScanRow
	if err := db.Raw(listQ, args).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return scanRowsToCollaborators(rows), total, nil
}

func instructorCandidatesBaseSQL() string {
	return userpicker.UserPickerSelectSQL(constants.TableAppUsers) + fmt.Sprintf(`
INNER JOIN %s ur ON ur.user_id = u.id
INNER JOIN %s ro ON ro.id = ur.role_id AND ro.name = @role_name
WHERE u.deleted_at IS NULL
  AND u.id NOT IN (
      SELECT cc.user_id
      FROM course_collaborators cc
      WHERE cc.course_id = @course_id AND cc.deleted_at IS NULL
  )`, constants.TableRBACUserRoles, constants.TableRBACRoles)
}

func (r *GormRepository) ListInstructorCandidates(ctx context.Context, courseID string, actorUserID string, filter domain.InstructorCandidateFilter) ([]domain.InstructorCandidate, int64, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := userpicker.ListRows(ctx, r.db, instructorCandidatesBaseSQL(), map[string]any{
		"course_id": courseID,
		"role_name": instructordomain.RoleNameInstructor,
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
