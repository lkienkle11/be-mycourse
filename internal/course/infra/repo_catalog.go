package infra

import (
	"context"

	"mycourse-io-be/internal/course/domain"
)

// trendingCoursesQuery reuses courseListOwnerDisplayNameColumn/courseListOwnerUserJoin
// (repos.go) rather than re-typing an equivalent join, matching ListPublishedCourses's
// query shape (repo_learner.go) but ordering by created_at instead of id (id is a
// random UUID, not actually recency order) and adding the owner-name join +
// short_description column that query doesn't select.
const trendingCoursesQuery = `
SELECT c.id, c.slug, c.created_at,
    pv.title, pv.short_description,
    COALESCE(pm.url, '') AS thumbnail_url,
    ` + courseListOwnerDisplayNameColumn + `
FROM courses c
INNER JOIN course_versions pv
    ON pv.id = c.current_published_version_id AND pv.deleted_at IS NULL
LEFT JOIN media_files pm
    ON pm.id = pv.thumbnail_file_id AND pm.deleted_at IS NULL
` + courseListOwnerUserJoin + `
WHERE c.deleted_at IS NULL AND c.trashed_at IS NULL AND c.current_published_version_id IS NOT NULL
ORDER BY c.created_at DESC
LIMIT ?`

type trendingCourseScanRow struct {
	ID               string `gorm:"column:id"`
	Slug             string `gorm:"column:slug"`
	CreatedAt        int64  `gorm:"column:created_at"`
	Title            string `gorm:"column:title"`
	ShortDescription string `gorm:"column:short_description"`
	ThumbnailURL     string `gorm:"column:thumbnail_url"`
	OwnerDisplayName string `gorm:"column:owner_display_name"`
}

func (row *trendingCourseScanRow) toDomain() domain.TrendingCourseItem {
	return domain.TrendingCourseItem{
		ID:               row.ID,
		Slug:             row.Slug,
		Title:            row.Title,
		ShortDescription: row.ShortDescription,
		ThumbnailURL:     row.ThumbnailURL,
		OwnerDisplayName: row.OwnerDisplayName,
		CreatedAt:        row.CreatedAt,
	}
}

func (r *GormRepository) ListTrendingCourses(ctx context.Context, limit int) ([]domain.TrendingCourseItem, error) {
	var rows []trendingCourseScanRow
	if err := r.db.WithContext(ctx).Raw(trendingCoursesQuery, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.TrendingCourseItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, nil
}
