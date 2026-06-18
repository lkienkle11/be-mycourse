package infra

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/timex"
	sharedutils "mycourse-io-be/internal/shared/utils"
)

func (r *GormRepository) ListPublishedCourses(ctx context.Context) ([]domain.CourseListItem, error) {
	q := `
SELECT
    ` + courseListBaseColumns + `,
    'LEARNER' AS role,
    pv.title,
    pv.status AS review_status,
    pv.version_no,
    TRUE AS has_published,
    (c.current_draft_version_id IS NOT NULL) AS has_draft,
    COALESCE(pv.thumbnail_file_id::text, '') AS thumbnail_file_id,
    COALESCE(pm.url, '') AS thumbnail_url,
    COALESCE(pv.preview_video_file_id::text, '') AS preview_video_file_id
FROM courses c
INNER JOIN course_versions pv
    ON pv.id = c.current_published_version_id AND pv.deleted_at IS NULL
LEFT JOIN media_files pm
    ON pm.id = pv.thumbnail_file_id AND pm.deleted_at IS NULL
WHERE c.deleted_at IS NULL
  AND c.trashed_at IS NULL
ORDER BY c.id DESC`
	var rows []courseListScanRow
	if err := r.db.WithContext(ctx).Raw(q).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CourseListItem, len(rows))
	for i, row := range rows {
		out[i] = toCourseListItem(&row)
	}
	return out, nil
}

func (r *GormRepository) GetLearningCourse(ctx context.Context, courseID string, userID string) (*domain.CourseDetail, error) {
	detail, err := r.loadLearnerCourseDetail(ctx, r.db.WithContext(ctx), courseID, userID)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (r *GormRepository) Enroll(ctx context.Context, courseID string, userID string) (*domain.Enrollment, error) {
	var out *domain.Enrollment
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		course, err := r.loadCourse(ctx, tx, courseID)
		if err != nil {
			return err
		}
		if course.CurrentPublishedVersionID == nil {
			return domain.ErrCoursePublishedRequired
		}
		var row enrollmentRow
		err = tx.Where("course_id = ? AND user_id = ? AND deleted_at IS NULL", courseID, userID).First(&row).Error
		if err == nil {
			enrollment := toEnrollment(&row)
			out = &enrollment
			return nil
		}
		if !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = enrollmentRow{CourseID: courseID, UserID: userID, CurrentVersionID: *course.CurrentPublishedVersionID}
		if err := touchCreateCourseEntity(ctx, tx, &row.CreatedAt, &row.UpdatedAt, &row); err != nil {
			return err
		}
		enrollment := toEnrollment(&row)
		out = &enrollment
		return nil
	})
	return out, err
}

func (r *GormRepository) GetProgress(ctx context.Context, courseID string, userID string) (*domain.CourseProgress, error) {
	return r.loadProgress(ctx, r.db.WithContext(ctx), courseID, userID)
}

func (r *GormRepository) SaveProgress(ctx context.Context, courseID string, userID string, in domain.SaveProgressInput) (*domain.CourseProgress, error) {
	var out *domain.CourseProgress
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		enrollment, err := r.requireEnrollment(ctx, tx, courseID, userID)
		if err != nil {
			return err
		}
		now := timex.NowUnix()
		var existing progressRow
		err = tx.Where("enrollment_id = ? AND stable_content_id = ? AND deleted_at IS NULL", enrollment.ID, in.StableContentID).
			First(&existing).Error
		if err == nil {
			if err := tx.Model(&progressRow{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"content_type":       strings.TrimSpace(in.ContentType),
				"status":             strings.TrimSpace(in.Status),
				"score":              in.Score,
				"quiz_attempt":       sharedutils.NormalizeJSON(in.QuizAttempt, "{}"),
				"last_interacted_at": now,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}
		} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else {
			row := &progressRow{
				EnrollmentID: enrollment.ID, StableContentID: in.StableContentID, ContentType: strings.TrimSpace(in.ContentType),
				Status: strings.TrimSpace(in.Status), Score: in.Score, QuizAttempt: sharedutils.NormalizeJSON(in.QuizAttempt, "{}"), LastInteractedAt: &now,
			}
			if err := touchCreateCourseEntity(ctx, tx, &row.CreatedAt, &row.UpdatedAt, row); err != nil {
				return err
			}
		}
		out, err = r.loadProgress(ctx, tx, courseID, userID)
		return err
	})
	return out, err
}
