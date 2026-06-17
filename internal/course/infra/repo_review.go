package infra

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/timex"
)

func (r *GormRepository) SubmitForReview(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	return r.updateDraftStatus(ctx, courseID, actorUserID, domain.VersionStatusDraft, domain.VersionStatusInReview, "", true)
}

func (r *GormRepository) ReopenDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := r.requireEditorAccess(ctx, tx, courseID, actorUserID); err != nil {
			return err
		}
		course, version, err := r.requireDraftVersion(ctx, tx, courseID)
		if err != nil {
			return err
		}
		if version.Status != domain.VersionStatusRejected {
			return domain.ErrCourseDraftRejectedOnly
		}
		newDraftID, err := r.createNextDraftVersion(ctx, tx, course.ID, &version.ID)
		if err != nil {
			return err
		}
		return tx.Model(&courseRow{}).Where("id = ?", course.ID).Updates(map[string]any{
			"current_draft_version_id": newDraftID,
			"updated_at":               timex.NowUnix(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true)
}

func (r *GormRepository) ListPendingReviews(ctx context.Context) ([]domain.CourseListItem, error) {
	q := `
SELECT
    ` + courseListBaseColumns + `,
    'OWNER' AS role,
    dv.title,
    dv.status AS review_status,
    dv.version_no,
    (c.current_published_version_id IS NOT NULL) AS has_published,
    TRUE AS has_draft,
    COALESCE(dv.thumbnail_file_id::text, '') AS thumbnail_file_id,
    COALESCE(dm.url, '') AS thumbnail_url,
    COALESCE(dv.preview_video_file_id::text, '') AS preview_video_file_id,
    dv.id::text AS version_id
FROM courses c
INNER JOIN course_versions dv
    ON dv.id = c.current_draft_version_id AND dv.status = @status AND dv.deleted_at IS NULL
LEFT JOIN media_files dm
    ON dm.id = dv.thumbnail_file_id AND dm.deleted_at IS NULL
WHERE c.deleted_at IS NULL
  AND c.trashed_at IS NULL
ORDER BY dv.updated_at DESC`
	var rows []courseListScanRow
	if err := r.db.WithContext(ctx).Raw(q, map[string]any{"status": domain.VersionStatusInReview}).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CourseListItem, len(rows))
	for i, row := range rows {
		out[i] = toCourseListItem(&row)
	}
	return out, nil
}

func (r *GormRepository) ApproveDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		course, version, err := r.requireDraftVersion(ctx, tx, courseID)
		if err != nil {
			return err
		}
		if version.Status != domain.VersionStatusInReview {
			return domain.ErrCourseInvalidReviewState
		}
		now := timex.NowUnix()
		if err := tx.Model(&courseVersionRow{}).Where("id = ?", version.ID).Updates(map[string]any{
			"status":              domain.VersionStatusApproved,
			"approved_by_user_id": actorUserID,
			"approved_at":         now,
			"updated_at":          now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&courseRow{}).Where("id = ?", course.ID).Updates(map[string]any{
			"current_published_version_id": version.ID,
			"current_draft_version_id":     nil,
			"updated_at":                   now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&enrollmentRow{}).Where("course_id = ? AND deleted_at IS NULL", course.ID).
			Updates(map[string]any{"current_version_id": version.ID, "updated_at": now}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true)
}

func (r *GormRepository) RejectDraft(ctx context.Context, courseID string, actorUserID string, reason string) (*domain.CourseDetail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		course, version, err := r.requireDraftVersion(ctx, tx, courseID)
		if err != nil {
			return err
		}
		if version.Status != domain.VersionStatusInReview {
			return domain.ErrCourseInvalidReviewState
		}
		now := timex.NowUnix()
		rejectedVersionID := version.ID
		if err := tx.Model(&courseVersionRow{}).Where("id = ?", rejectedVersionID).Updates(map[string]any{
			"status":              domain.VersionStatusRejected,
			"rejected_by_user_id": actorUserID,
			"rejected_at":         now,
			"rejection_reason":    strings.TrimSpace(reason),
			"updated_at":          now,
		}).Error; err != nil {
			return err
		}
		newDraftID, err := r.createNextDraftVersion(ctx, tx, course.ID, &rejectedVersionID)
		if err != nil {
			return err
		}
		return tx.Model(&courseRow{}).Where("id = ?", course.ID).Updates(map[string]any{
			"current_draft_version_id": newDraftID,
			"updated_at":               now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true)
}
