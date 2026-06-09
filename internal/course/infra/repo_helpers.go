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

type versionRefValidationInput struct {
	ThumbnailFileID    *string
	PreviewVideoFileID *string
	CourseLevelID      *string
	CourseTopicID      *string
	TagIDs             []string
	SkillIDs           []string
	OutcomeIDs         []string
}

type draftEntityUpdateConfig[T any, D any] struct {
	EntityID           *string
	ExpectedRowVersion int64
	Load               func(context.Context, *gorm.DB, string, string) (*T, error)
	Behavior           draftEntityBehavior[T, D]
	Updates            map[string]any
}

type draftEntityBehavior[T any, D any] struct {
	RowVersion func(*T) int64
	RowID      func(*T) string
	Model      any
	ToDomain   func(*T) D
}

type draftOutlineDeleteConfig struct {
	Resolve func(context.Context, *gorm.DB, string, string) (string, error)
	Remove  func(context.Context, *gorm.DB, string) error
}

type draftEditScope struct {
	CourseID    string
	ActorUserID string
}

var sectionBehavior = draftEntityBehavior[sectionRow, domain.Section]{
	RowVersion: sectionRowVersion,
	RowID:      sectionRowID,
	Model:      &sectionRow{},
	ToDomain:   toSection,
}

var lessonBehavior = draftEntityBehavior[lessonRow, domain.Lesson]{
	RowVersion: lessonRowVersion,
	RowID:      lessonRowID,
	Model:      &lessonRow{},
	ToDomain:   toLesson,
}

func loadActiveRow[T any](ctx context.Context, db *gorm.DB, notFound error, query string, args ...any) (*T, error) {
	var row T
	if err := db.WithContext(ctx).Where(query, args...).First(&row).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound
		}
		return nil, err
	}
	return &row, nil
}

func loadActiveRows[T any](ctx context.Context, db *gorm.DB, query string, args ...any) ([]T, error) {
	var rows []T
	if err := db.WithContext(ctx).Where(query, args...).Order("order_index ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func optimisticUpdate(ctx context.Context, tx *gorm.DB, model any, rowID string, expectedRowVersion int64, updates map[string]any) error {
	updates["updated_at"] = timex.NowUnix()
	updates["row_version"] = gorm.Expr("row_version + 1")
	result := tx.WithContext(ctx).
		Model(model).
		Where("id = ? AND row_version = ? AND deleted_at IS NULL", rowID, expectedRowVersion).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrCourseOptimisticLock
	}
	return nil
}

func updateDraftEntity[T any, D any](
	r *GormRepository,
	ctx context.Context,
	courseID string, actorUserID string,
	cfg draftEntityUpdateConfig[T, D],
) (*D, error) {
	if cfg.EntityID == nil {
		return nil, domain.ErrCourseNotFound
	}

	var out *D
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		row, err := cfg.Load(ctx, tx, *cfg.EntityID, *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if cfg.Behavior.RowVersion(row) != cfg.ExpectedRowVersion {
			return domain.ErrCourseOptimisticLock
		}
		if err := optimisticUpdate(ctx, tx, cfg.Behavior.Model, cfg.Behavior.RowID(row), cfg.ExpectedRowVersion, cfg.Updates); err != nil {
			return err
		}
		fresh, err := cfg.Load(ctx, tx, cfg.Behavior.RowID(row), *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		value := cfg.Behavior.ToDomain(fresh)
		out = &value
		return nil
	})
	return out, err
}

func updateOutlineEntity[T any, D any](
	r *GormRepository,
	ctx context.Context,
	scope draftEditScope,
	entityID *string,
	expectedRowVersion int64,
	load func(context.Context, *gorm.DB, string, string) (*T, error),
	behavior draftEntityBehavior[T, D],
	updates map[string]any,
) (*D, error) {
	return updateDraftEntity(r, ctx, scope.CourseID, scope.ActorUserID, draftEntityUpdateConfig[T, D]{
		EntityID:           entityID,
		ExpectedRowVersion: expectedRowVersion,
		Load:               load,
		Behavior:           behavior,
		Updates:            updates,
	})
}

func (r *GormRepository) deleteDraftOutline(
	ctx context.Context,
	courseID string, actorUserID string, resourceID string,
	cfg draftOutlineDeleteConfig,
) ([]domain.Section, error) {
	var outline []domain.Section
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		resolvedID, err := cfg.Resolve(ctx, tx, resourceID, *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if err := cfg.Remove(ctx, tx, resolvedID); err != nil {
			return err
		}
		outline, err = r.loadOutline(ctx, tx, *access.CurrentDraftVersionID)
		return err
	})
	return outline, err
}

func buildBasicInfoUpdates(in domain.UpdateBasicInfoInput) map[string]any {
	updates := map[string]any{
		"updated_at":  timex.NowUnix(),
		"row_version": gorm.Expr("row_version + 1"),
	}
	if in.Title != nil {
		updates["title"] = strings.TrimSpace(*in.Title)
	}
	if in.ShortDescription != nil {
		updates["short_description"] = strings.TrimSpace(*in.ShortDescription)
	}
	if in.AboutCourse != nil {
		updates["about_course"] = strings.TrimSpace(*in.AboutCourse)
	}
	if in.ThumbnailFileID != nil {
		updates["thumbnail_file_id"] = sharedutils.NilIfBlank(*in.ThumbnailFileID)
	}
	if in.PreviewVideoFileID != nil {
		updates["preview_video_file_id"] = sharedutils.NilIfBlank(*in.PreviewVideoFileID)
	}
	if in.CourseLevelID != nil {
		updates["course_level_id"] = sharedutils.NilIfBlank(*in.CourseLevelID)
	}
	if in.CourseTopicID != nil {
		updates["course_topic_id"] = sharedutils.NilIfBlank(*in.CourseTopicID)
	}
	return updates
}

func sectionUpdates(in domain.UpsertSectionInput) map[string]any {
	return map[string]any{
		"title":       strings.TrimSpace(in.Title),
		"description": strings.TrimSpace(in.Description),
	}
}

func lessonUpdates(in domain.UpsertLessonInput) map[string]any {
	return map[string]any{
		"title":   strings.TrimSpace(in.Title),
		"summary": strings.TrimSpace(in.Summary),
	}
}

func sectionRowVersion(row *sectionRow) int64 { return row.RowVersion }

func lessonRowVersion(row *lessonRow) int64 { return row.RowVersion }

func sectionRowID(row *sectionRow) string { return row.ID }

func lessonRowID(row *lessonRow) string { return row.ID }

func (r *GormRepository) resolveSectionID(ctx context.Context, tx *gorm.DB, id, versionID string) (string, error) {
	row, err := r.loadSection(ctx, tx, id, versionID)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func (r *GormRepository) resolveLessonID(ctx context.Context, tx *gorm.DB, id, versionID string) (string, error) {
	row, err := r.loadLesson(ctx, tx, id, versionID)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func (r *GormRepository) deleteSectionTree(ctx context.Context, tx *gorm.DB, rowID string) error {
	return deleteChildrenThenRow(ctx, tx, "section_id", &lessonRow{}, &sectionRow{}, rowID)
}

func (r *GormRepository) deleteLessonTree(ctx context.Context, tx *gorm.DB, rowID string) error {
	return deleteChildrenThenRow(ctx, tx, "lesson_id", &subLessonRow{}, &lessonRow{}, rowID)
}

func deleteChildrenThenRow(ctx context.Context, tx *gorm.DB, column string, childModel, model any, rowID string) error {
	if err := tx.WithContext(ctx).Where(column+" = ?", rowID).Delete(childModel).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Delete(model, rowID).Error
}
