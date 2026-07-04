package infra

import (
	"context"
	"fmt"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/userpicker"
)

func sqlUserWithoutPlatformStaffRoles(userIDColumn string) string {
	return fmt.Sprintf(`
  AND NOT EXISTS (
      SELECT 1
      FROM %s ur_staff
      INNER JOIN %s ro_staff ON ro_staff.id = ur_staff.role_id AND ro_staff.name IN ('%s', '%s')
      WHERE ur_staff.user_id = %s
  )`, constants.TableRBACUserRoles, constants.TableRBACRoles, domain.RoleNameSysadmin, domain.RoleNameAdmin, userIDColumn)
}

func sqlUserWithoutRosterBlockingRoles(userIDColumn string) string {
	return fmt.Sprintf(`
  AND NOT EXISTS (
      SELECT 1
      FROM %s ur
      INNER JOIN %s ro ON ro.id = ur.role_id AND ro.name IN ('%s', '%s', '%s')
      WHERE ur.user_id = %s
  )`, constants.TableRBACUserRoles, constants.TableRBACRoles, domain.RoleNameInstructor, domain.RoleNameSysadmin, domain.RoleNameAdmin, userIDColumn)
}

func rosterCandidatesBaseSQL() string {
	return userpicker.UserPickerSelectSQL(constants.TableAppUsers) + fmt.Sprintf(`
WHERE u.deleted_at IS NULL%s%s`, sqlUserWithoutRosterBlockingRoles("u.id"), userpicker.EligiblePickerWhereClause())
}

func (r *GormRepository) ListRoster(ctx context.Context, f domain.RosterFilter) ([]domain.RosterMember, int64, error) {
	now := timex.NowUnix()
	base := fmt.Sprintf(`
SELECT u.id, u.display_name, u.email, COALESCE(u.phone, '') AS phone,
       COALESCE(u.avatar_file_id::text, '') AS avatar_file_id
FROM %s u
INNER JOIN %s ur ON ur.user_id = u.id
INNER JOIN %s ro ON ro.id = ur.role_id AND ro.name = @role_name
WHERE u.deleted_at IS NULL%s%s`, constants.TableAppUsers, constants.TableRBACUserRoles, constants.TableRBACRoles,
		sqlUserWithoutPlatformStaffRoles("u.id"), userpicker.ActiveUserWhereClause())
	q := r.db.WithContext(ctx).Table("(?) AS roster", r.db.Raw(base, map[string]any{
		"role_name": domain.RoleNameInstructor,
		"now":       now,
	}))
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("display_name ILIKE ? OR email ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := instrPageParams(f.Page, f.PageSize)
	type scanRow struct {
		ID           string
		DisplayName  string
		Email        string
		Phone        string
		AvatarFileID string
	}
	var rows []scanRow
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.RosterMember, len(rows))
	for i, row := range rows {
		out[i] = domain.RosterMember{
			UserID: row.ID, FullName: row.DisplayName, Email: row.Email,
			Phone: row.Phone, AvatarFileID: row.AvatarFileID,
		}
	}
	return out, total, nil
}

func (r *GormRepository) ListRosterCandidates(ctx context.Context, f domain.RosterCandidateFilter) ([]domain.RosterCandidate, int64, error) {
	rows, total, err := userpicker.ListRows(ctx, r.db, rosterCandidatesBaseSQL(), map[string]any{
		"now": timex.NowUnix(),
	}, userpicker.ListFilter{Page: f.Page, PerPage: f.PageSize, Search: f.Search})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.RosterCandidate, len(rows))
	for i, row := range rows {
		out[i] = domain.RosterCandidate{
			UserID: row.UserID, DisplayName: row.DisplayName, Email: row.Email,
			AvatarFileID: row.AvatarFileID, AvatarURL: row.AvatarURL,
		}
	}
	return out, total, nil
}
