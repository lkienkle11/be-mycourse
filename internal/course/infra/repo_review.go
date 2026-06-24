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
		if _, err := r.requireOwnerAccess(ctx, tx, courseID, actorUserID); err != nil {
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
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true, true)
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

func (r *GormRepository) ApproveDraft(ctx context.Context, courseID string, actorUserID string, approvalNote string) (*domain.CourseDetail, error) {
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
			"approval_note":       strings.TrimSpace(approvalNote),
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
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true, true)
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
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true, true)
}

type reviewHistoryScanRow struct {
	VersionNo  int    `gorm:"column:version_no"`
	Status     string `gorm:"column:status"`
	Note       string `gorm:"column:note"`
	ReviewedAt int64  `gorm:"column:reviewed_at"`
}

func reviewHistoryStatusFilter(status string) string {
	return strings.TrimSpace(status)
}

func reviewHistoryPagination(filter domain.ReviewHistoryFilter) (page, perPage, offset int) {
	page = filter.Page
	if page < 1 {
		page = 1
	}
	perPage = filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	return page, perPage, (page - 1) * perPage
}

func (r *GormRepository) countReviewHistoryRows(ctx context.Context, courseID, statusFilter string) (int64, error) {
	countQ := `
SELECT COUNT(*)::bigint
FROM course_versions
WHERE course_id = @courseId
  AND deleted_at IS NULL
  AND status IN (@approved, @rejected)
  AND (@status = '' OR status = @status)`
	var total int64
	err := r.db.WithContext(ctx).Raw(countQ, map[string]any{
		"courseId": courseID,
		"approved": domain.VersionStatusApproved,
		"rejected": domain.VersionStatusRejected,
		"status":   statusFilter,
	}).Scan(&total).Error
	return total, err
}

func (r *GormRepository) loadReviewHistoryRows(ctx context.Context, courseID, statusFilter string, limit, offset int) ([]reviewHistoryScanRow, error) {
	listQ := `
SELECT
    version_no,
    status,
    CASE
        WHEN status = @approved THEN approval_note
        ELSE rejection_reason
    END AS note,
    COALESCE(approved_at, rejected_at, 0) AS reviewed_at
FROM course_versions
WHERE course_id = @courseId
  AND deleted_at IS NULL
  AND status IN (@approved, @rejected)
  AND (@status = '' OR status = @status)
ORDER BY COALESCE(approved_at, rejected_at) DESC
LIMIT @limit OFFSET @offset`
	var rows []reviewHistoryScanRow
	err := r.db.WithContext(ctx).Raw(listQ, map[string]any{
		"courseId": courseID,
		"approved": domain.VersionStatusApproved,
		"rejected": domain.VersionStatusRejected,
		"status":   statusFilter,
		"limit":    limit,
		"offset":   offset,
	}).Scan(&rows).Error
	return rows, err
}

func (r *GormRepository) ListReviewHistory(ctx context.Context, courseID string, actorUserID string, filter domain.ReviewHistoryFilter) ([]domain.CourseReviewHistoryItem, int64, error) {
	if _, err := r.requireEditorAccess(ctx, r.db, courseID, actorUserID); err != nil {
		return nil, 0, err
	}

	statusFilter := reviewHistoryStatusFilter(filter.Status)
	total, err := r.countReviewHistoryRows(ctx, courseID, statusFilter)
	if err != nil {
		return nil, 0, err
	}

	_, perPage, offset := reviewHistoryPagination(filter)
	rows, err := r.loadReviewHistoryRows(ctx, courseID, statusFilter, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	items := make([]domain.CourseReviewHistoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.CourseReviewHistoryItem{
			VersionNo: row.VersionNo, Status: row.Status, Note: row.Note, ReviewedAt: row.ReviewedAt,
		})
	}
	return items, total, nil
}
