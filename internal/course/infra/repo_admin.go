package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/timex"
)

const adminCoursePublishedListSelect = `
    ` + courseListBaseColumns + `,
    ` + courseListOwnerDisplayNameColumn + `,
    'ADMIN' AS role,
    pv.title AS title,
    pv.status AS review_status,
    pv.id::text AS version_id,
    pv.version_no AS version_no,
    true AS has_published,
    (c.current_draft_version_id IS NOT NULL) AS has_draft,
    COALESCE(pv.thumbnail_file_id::text, '') AS thumbnail_file_id,
    COALESCE(pm.url, '') AS thumbnail_url,
    COALESCE(pv.preview_video_file_id::text, '') AS preview_video_file_id,
    COALESCE(dv.status, '') AS draft_review_status`

const adminCoursePublishedListJoins = `
FROM courses c` + courseListOwnerUserJoin + `
INNER JOIN course_versions pv
    ON pv.id = c.current_published_version_id AND pv.deleted_at IS NULL
LEFT JOIN course_versions dv
    ON dv.id = c.current_draft_version_id AND dv.deleted_at IS NULL
LEFT JOIN media_files pm
    ON pm.id = pv.thumbnail_file_id AND pm.deleted_at IS NULL`

func (r *GormRepository) ListAdminCourses(ctx context.Context) ([]domain.CourseListItem, error) {
	q := `
SELECT` + adminCoursePublishedListSelect + adminCoursePublishedListJoins + `
WHERE c.deleted_at IS NULL
  AND c.trashed_at IS NULL
  AND pv.status = @approved_status
ORDER BY c.updated_at DESC`

	var rows []courseListScanRow
	if err := r.db.WithContext(ctx).Raw(q, map[string]any{
		"approved_status": domain.VersionStatusApproved,
	}).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return mapCourseListScanRows(rows), nil
}

func (r *GormRepository) ListTrashedCourses(ctx context.Context) ([]domain.CourseListItem, error) {
	q := `
SELECT` + adminCoursePublishedListSelect + adminCoursePublishedListJoins + `
WHERE c.deleted_at IS NULL
  AND c.trashed_at IS NOT NULL
  AND pv.status = @approved_status
  AND (dv.id IS NULL OR dv.status <> @rejected_status)
ORDER BY c.trashed_at DESC`

	var rows []courseListScanRow
	if err := r.db.WithContext(ctx).Raw(q, map[string]any{
		"approved_status": domain.VersionStatusApproved,
		"rejected_status": domain.VersionStatusRejected,
	}).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return mapCourseListScanRows(rows), nil
}

func (r *GormRepository) TrashCourse(ctx context.Context, courseID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		course, err := r.loadCourseForAdmin(ctx, tx, courseID)
		if err != nil {
			return err
		}
		if course.TrashedAt != nil {
			return domain.ErrCourseTrashed
		}
		published, draft, err := r.loadPublishedAndDraftVersions(ctx, tx, course)
		if err != nil {
			return err
		}
		if !courseEligibleForTrash(published, draft) {
			return domain.ErrCourseTrashNotEligible
		}
		now := timex.NowUnix()
		return tx.Model(&courseRow{}).Where("id = ? AND deleted_at IS NULL", course.ID).
			Updates(map[string]any{"trashed_at": now, "updated_at": now}).Error
	})
}

func (r *GormRepository) RestoreCourse(ctx context.Context, courseID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		course, err := r.loadTrashedCourse(ctx, tx, courseID)
		if err != nil {
			return err
		}
		now := timex.NowUnix()
		return tx.Model(&courseRow{}).Where("id = ?", course.ID).
			Updates(map[string]any{"trashed_at": nil, "updated_at": now}).Error
	})
}

func (r *GormRepository) PermanentDeleteCourse(ctx context.Context, courseID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		course, err := r.loadTrashedCourse(ctx, tx, courseID)
		if err != nil {
			return err
		}
		return r.softDeleteCourseTree(ctx, tx, course.ID)
	})
}

func (r *GormRepository) loadPublishedAndDraftVersions(
	ctx context.Context,
	tx *gorm.DB,
	course *courseRow,
) (published, draft *courseVersionRow, err error) {
	if course.CurrentPublishedVersionID != nil {
		published, err = r.loadVersionRow(ctx, tx, *course.CurrentPublishedVersionID)
		if err != nil {
			return nil, nil, err
		}
	}
	if course.CurrentDraftVersionID != nil {
		draft, err = r.loadVersionRow(ctx, tx, *course.CurrentDraftVersionID)
		if err != nil {
			return nil, nil, err
		}
	}
	return published, draft, nil
}

func courseEligibleForTrash(published, draft *courseVersionRow) bool {
	if published == nil || published.Status != domain.VersionStatusApproved {
		return false
	}
	if draft != nil && draft.Status == domain.VersionStatusRejected {
		return false
	}
	return true
}

func mapCourseListScanRows(rows []courseListScanRow) []domain.CourseListItem {
	out := make([]domain.CourseListItem, len(rows))
	for i, row := range rows {
		out[i] = toCourseListItem(&row)
	}
	return out
}

func (r *GormRepository) softDeleteCourseTree(ctx context.Context, tx *gorm.DB, courseID string) error {
	now := timex.NowUnix()
	for _, model := range []any{&courseRow{}, &courseVersionRow{}, &collaboratorRow{}, &enrollmentRow{}, &progressRow{}} {
		switch model.(type) {
		case *courseRow:
			if err := tx.Model(model).Where("id = ? AND deleted_at IS NULL", courseID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
		case *courseVersionRow:
			if err := tx.Model(model).Where("course_id = ? AND deleted_at IS NULL", courseID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
		case *collaboratorRow:
			if err := tx.Model(model).Where("course_id = ? AND deleted_at IS NULL", courseID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
		case *enrollmentRow:
			if err := tx.Model(model).Where("course_id = ? AND deleted_at IS NULL", courseID).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
		case *progressRow:
			if err := tx.Exec(`
UPDATE course_progress_items
SET deleted_at = ?, updated_at = ?
WHERE deleted_at IS NULL
  AND enrollment_id IN (
      SELECT id FROM course_enrollments
      WHERE course_id = ? AND deleted_at IS NULL
  )`, now, now, courseID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
