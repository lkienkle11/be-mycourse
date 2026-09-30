package infra

import (
	"context"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/userpicker"
)

// popularInstructorsQuery matches ListRoster's (repo_roster_list.go) "is this
// user a currently-active instructor" check — reuses userpicker.ActiveUserWhereClause()
// (is_disable + banned_until) instead of hand-rolling a weaker is_disable-only
// check. The courses aggregate is computed once in a subquery, then joined to
// identity/profile data — not per-instructor N+1 (.ai/skills/logic-n-1-optimize).
var popularInstructorsQuery = `
SELECT u.id AS user_id, u.display_name,
    COALESCE(m.url, '') AS avatar_url,
    COALESCE(ip.current_job_title, '') AS subtitle,
    agg.course_count
FROM (
    SELECT c.owner_user_id, COUNT(*) AS course_count, MAX(c.created_at) AS latest_course_at
    FROM ` + constants.TableCourses + ` c
    WHERE c.deleted_at IS NULL AND c.trashed_at IS NULL AND c.current_published_version_id IS NOT NULL
    GROUP BY c.owner_user_id
) agg
INNER JOIN ` + constants.TableAppUsers + ` u ON u.id = agg.owner_user_id AND u.deleted_at IS NULL
INNER JOIN ` + constants.TableRBACUserRoles + ` ur ON ur.user_id = u.id
INNER JOIN ` + constants.TableRBACRoles + ` r ON r.id = ur.role_id AND r.name = @role_name
LEFT JOIN ` + constants.TableInstructorProfiles + ` ip ON ip.user_id = u.id AND ip.deleted_at IS NULL
LEFT JOIN ` + constants.TableMediaFiles + ` m ON m.id = u.avatar_file_id AND m.deleted_at IS NULL
WHERE 1=1 ` + userpicker.ActiveUserWhereClause() + `
ORDER BY agg.course_count DESC, agg.latest_course_at DESC
LIMIT @limit`

type popularInstructorScanRow struct {
	UserID      string `gorm:"column:user_id"`
	DisplayName string `gorm:"column:display_name"`
	AvatarURL   string `gorm:"column:avatar_url"`
	Subtitle    string `gorm:"column:subtitle"`
	CourseCount int64  `gorm:"column:course_count"`
}

func (row *popularInstructorScanRow) toDomain() domain.PopularInstructor {
	return domain.PopularInstructor{
		UserID:      row.UserID,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarURL,
		Subtitle:    row.Subtitle,
		CourseCount: row.CourseCount,
	}
}

func (r *GormRepository) ListPopularInstructors(ctx context.Context, limit int) ([]domain.PopularInstructor, error) {
	var rows []popularInstructorScanRow
	err := r.db.WithContext(ctx).Raw(popularInstructorsQuery, map[string]any{
		"role_name": domain.RoleNameInstructor,
		"now":       timex.NowUnix(),
		"limit":     limit,
	}).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]domain.PopularInstructor, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, nil
}
