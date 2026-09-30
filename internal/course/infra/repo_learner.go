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

// continueLearningQuery counts ALL sub-lessons (VIDEO + QUIZ + TEXT alike) per
// the course outline hierarchy (Section -> Lesson -> Sub-lesson) — not just
// VIDEO kind — per product decision (openspec/changes/add-home-catalog-apis):
// the Figma mock's "X/Y Videos Completed" label only showed video-kind
// courses; this project's actual courses mix VIDEO/QUIZ/TEXT sub-lessons
// under one course, so progress counts every sub-lesson regardless of kind.
// Completion is derived from course_sub_lessons existing (authoritative
// outline) joined to course_progress_items by stable_id, not from the
// client-supplied, unvalidated course_progress_items.content_type column
// (dto.go's saveProgressRequest only has validate:"required"; CourseService.
// SaveProgress is a pure passthrough to the repo). See design.md Decision #5 / Risks.
const continueLearningQuery = `
SELECT c.id, c.slug, cv.title,
    COALESCE(pm.url, '') AS thumbnail_url,
    COALESCE(ou.display_name, '') AS owner_display_name,
    COALESCE(prog2.completed_sub_lessons, 0) AS completed_sub_lessons,
    COALESCE(prog2.total_sub_lessons, 0) AS total_sub_lessons,
    COALESCE(prog.last_interacted_at, e.created_at) AS last_activity_at
FROM course_enrollments e
INNER JOIN courses c ON c.id = e.course_id AND c.deleted_at IS NULL
INNER JOIN course_versions cv ON cv.id = e.current_version_id AND cv.deleted_at IS NULL
LEFT JOIN media_files pm ON pm.id = cv.thumbnail_file_id AND pm.deleted_at IS NULL
LEFT JOIN users ou ON ou.id = c.owner_user_id AND ou.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT MAX(last_interacted_at) AS last_interacted_at
    FROM course_progress_items
    WHERE enrollment_id = e.id AND deleted_at IS NULL
) prog ON TRUE
LEFT JOIN LATERAL (
    SELECT
        COUNT(*) AS total_sub_lessons,
        COUNT(*) FILTER (WHERE pi.status = 'COMPLETED') AS completed_sub_lessons
    FROM course_sub_lessons sl
    LEFT JOIN course_progress_items pi
        ON pi.enrollment_id = e.id
       AND pi.stable_content_id = sl.stable_id
       AND pi.deleted_at IS NULL
    WHERE sl.course_version_id = e.current_version_id
      AND sl.deleted_at IS NULL
) prog2 ON TRUE
WHERE e.user_id = ? AND e.deleted_at IS NULL
ORDER BY COALESCE(prog.last_interacted_at, e.created_at) DESC
LIMIT ?`

type continueLearningScanRow struct {
	ID                  string `gorm:"column:id"`
	Slug                string `gorm:"column:slug"`
	Title               string `gorm:"column:title"`
	ThumbnailURL        string `gorm:"column:thumbnail_url"`
	OwnerDisplayName    string `gorm:"column:owner_display_name"`
	CompletedSubLessons int    `gorm:"column:completed_sub_lessons"`
	TotalSubLessons     int    `gorm:"column:total_sub_lessons"`
	LastActivityAt      int64  `gorm:"column:last_activity_at"`
}

func (row *continueLearningScanRow) toDomain() domain.ContinueLearningItem {
	percent := 0.0
	if row.TotalSubLessons > 0 {
		percent = float64(row.CompletedSubLessons) / float64(row.TotalSubLessons) * 100
	}
	return domain.ContinueLearningItem{
		CourseID:            row.ID,
		Slug:                row.Slug,
		Title:               row.Title,
		ThumbnailURL:        row.ThumbnailURL,
		OwnerDisplayName:    row.OwnerDisplayName,
		CompletedSubLessons: row.CompletedSubLessons,
		TotalSubLessons:     row.TotalSubLessons,
		ProgressPercent:     percent,
		LastActivityAt:      row.LastActivityAt,
	}
}

func (r *GormRepository) ListContinueLearning(ctx context.Context, userID string, limit int) ([]domain.ContinueLearningItem, error) {
	var rows []continueLearningScanRow
	if err := r.db.WithContext(ctx).Raw(continueLearningQuery, userID, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ContinueLearningItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, nil
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
