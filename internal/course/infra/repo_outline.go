package infra

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/timex"
)

func (r *GormRepository) CreateSection(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSectionInput) (*domain.Section, error) {
	var out *domain.Section
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		next, err := r.nextOrderIndex(ctx, tx, constants.TableCourseSections, "course_version_id = ? AND deleted_at IS NULL", *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		row := &sectionRow{
			StableID: uuid.NewString(), CourseVersionID: *access.CurrentDraftVersionID,
			Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description), OrderIndex: next, RowVersion: 1,
		}
		if err := touchCreateCourseEntity(ctx, tx, &row.CreatedAt, &row.UpdatedAt, row); err != nil {
			return err
		}
		sec := toSection(row)
		out = &sec
		return nil
	})
	return out, err
}

func (r *GormRepository) UpdateSection(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSectionInput) (*domain.Section, error) {
	return updateSectionBySpec(
		r,
		ctx,
		draftEditScope{CourseID: courseID, ActorUserID: actorUserID},
		in.SectionID,
		in.ExpectedRowVersion,
		sectionUpdates(in),
	)
}

func (r *GormRepository) DeleteSection(ctx context.Context, courseID string, actorUserID string, sectionID string) ([]domain.Section, error) {
	return r.deleteDraftOutline(ctx, courseID, actorUserID, sectionID, draftOutlineDeleteConfig{
		Resolve: r.resolveSectionID,
		Remove:  r.deleteSectionTree,
	})
}

func (r *GormRepository) ReorderSections(ctx context.Context, courseID string, actorUserID string, orderedStableIDs []string) ([]domain.Section, error) {
	var outline []domain.Section
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		rows, err := r.loadSectionsByVersion(ctx, tx, *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if !sameStableIDs(rows, orderedStableIDs, func(row sectionRow) string { return row.StableID }) {
			return domain.ErrCourseInvalidOrdering
		}
		if err := reorderStableIDRows(ctx, tx, &sectionRow{}, "course_version_id = ?", *access.CurrentDraftVersionID, orderedStableIDs); err != nil {
			return err
		}
		outline, err = r.loadOutline(ctx, tx, *access.CurrentDraftVersionID)
		return err
	})
	return outline, err
}

func (r *GormRepository) CreateLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertLessonInput) (*domain.Lesson, error) {
	var out *domain.Lesson
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		if _, err := r.loadSection(ctx, tx, in.SectionID, *access.CurrentDraftVersionID); err != nil {
			return err
		}
		next, err := r.nextOrderIndex(ctx, tx, constants.TableCourseLessons, "section_id = ? AND deleted_at IS NULL", in.SectionID)
		if err != nil {
			return err
		}
		row := &lessonRow{
			StableID: uuid.NewString(), CourseVersionID: *access.CurrentDraftVersionID, SectionID: in.SectionID,
			Title: strings.TrimSpace(in.Title), Summary: strings.TrimSpace(in.Summary), OrderIndex: next, RowVersion: 1,
		}
		if err := touchCreateCourseEntity(ctx, tx, &row.CreatedAt, &row.UpdatedAt, row); err != nil {
			return err
		}
		lesson := toLesson(row)
		out = &lesson
		return nil
	})
	return out, err
}

func (r *GormRepository) UpdateLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertLessonInput) (*domain.Lesson, error) {
	return updateLessonBySpec(
		r,
		ctx,
		draftEditScope{CourseID: courseID, ActorUserID: actorUserID},
		in.LessonID,
		in.ExpectedRowVersion,
		lessonUpdates(in),
	)
}

func (r *GormRepository) DeleteLesson(ctx context.Context, courseID string, actorUserID string, lessonID string) ([]domain.Section, error) {
	return r.deleteDraftOutline(ctx, courseID, actorUserID, lessonID, draftOutlineDeleteConfig{
		Resolve: r.resolveLessonID,
		Remove:  r.deleteLessonTree,
	})
}

func (r *GormRepository) ReorderLessons(ctx context.Context, courseID string, actorUserID string, sectionID string, orderedStableIDs []string) ([]domain.Lesson, error) {
	var out []domain.Lesson
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		if _, err := r.loadSection(ctx, tx, sectionID, *access.CurrentDraftVersionID); err != nil {
			return err
		}
		rows, err := r.loadLessonsBySection(ctx, tx, sectionID)
		if err != nil {
			return err
		}
		if !sameStableIDs(rows, orderedStableIDs, func(row lessonRow) string { return row.StableID }) {
			return domain.ErrCourseInvalidOrdering
		}
		if err := reorderStableIDRows(ctx, tx, &lessonRow{}, "section_id = ?", sectionID, orderedStableIDs); err != nil {
			return err
		}
		rows, err = r.loadLessonsBySection(ctx, tx, sectionID)
		if err != nil {
			return err
		}
		out = make([]domain.Lesson, len(rows))
		for i := range rows {
			out[i] = toLesson(&rows[i])
		}
		return nil
	})
	return out, err
}

func (r *GormRepository) CreateSubLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSubLessonInput) (*domain.SubLesson, error) {
	var out *domain.SubLesson
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		if _, err := r.loadLesson(ctx, tx, in.LessonID, *access.CurrentDraftVersionID); err != nil {
			return err
		}
		if err := r.validateSubLessonPayload(ctx, tx, in); err != nil {
			return err
		}
		next, err := r.nextOrderIndex(ctx, tx, constants.TableCourseSubLessons, "lesson_id = ? AND deleted_at IS NULL", in.LessonID)
		if err != nil {
			return err
		}
		row := &subLessonRow{
			StableID: uuid.NewString(), CourseVersionID: *access.CurrentDraftVersionID, LessonID: in.LessonID,
			Title: strings.TrimSpace(in.Title), Kind: strings.ToUpper(strings.TrimSpace(in.Kind)), IsPreview: in.IsPreview,
			OrderIndex: next, RowVersion: 1,
		}
		if err := touchCreateCourseEntity(ctx, tx, &row.CreatedAt, &row.UpdatedAt, row); err != nil {
			return err
		}
		if err := r.upsertSubLessonDetail(ctx, tx, row.ID, in); err != nil {
			return err
		}
		sub, err := r.loadSubLessonDomain(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		out = sub
		return nil
	})
	return out, err
}

func (r *GormRepository) UpdateSubLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSubLessonInput) (*domain.SubLesson, error) {
	if in.SubLessonID == nil {
		return nil, domain.ErrCourseNotFound
	}
	var out *domain.SubLesson
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		row, err := r.loadSubLesson(ctx, tx, *in.SubLessonID, *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if row.RowVersion != in.ExpectedRowVersion {
			return domain.ErrCourseOptimisticLock
		}
		if err := r.validateSubLessonPayload(ctx, tx, in); err != nil {
			return err
		}
		result := tx.Model(&subLessonRow{}).
			Where("id = ? AND row_version = ? AND deleted_at IS NULL", row.ID, in.ExpectedRowVersion).
			Updates(map[string]any{
				"title":       strings.TrimSpace(in.Title),
				"kind":        strings.ToUpper(strings.TrimSpace(in.Kind)),
				"is_preview":  in.IsPreview,
				"updated_at":  timex.NowUnix(),
				"row_version": gorm.Expr("row_version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrCourseOptimisticLock
		}
		if err := r.deleteSubLessonDetails(ctx, tx, row.ID); err != nil {
			return err
		}
		if err := r.upsertSubLessonDetail(ctx, tx, row.ID, in); err != nil {
			return err
		}
		sub, err := r.loadSubLessonDomain(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		out = sub
		return nil
	})
	return out, err
}

func (r *GormRepository) DeleteSubLesson(ctx context.Context, courseID string, actorUserID string, subLessonID string) ([]domain.Section, error) {
	var outline []domain.Section
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		row, err := r.loadSubLesson(ctx, tx, subLessonID, *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if err := r.deleteSubLessonDetails(ctx, tx, row.ID); err != nil {
			return err
		}
		if err := tx.Delete(&subLessonRow{}, row.ID).Error; err != nil {
			return err
		}
		outline, err = r.loadOutline(ctx, tx, *access.CurrentDraftVersionID)
		return err
	})
	return outline, err
}

func (r *GormRepository) ReorderSubLessons(ctx context.Context, courseID string, actorUserID string, lessonID string, orderedStableIDs []string) ([]domain.SubLesson, error) {
	var out []domain.SubLesson
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		if _, err := r.loadLesson(ctx, tx, lessonID, *access.CurrentDraftVersionID); err != nil {
			return err
		}
		rows, err := r.loadSubLessonsByLesson(ctx, tx, lessonID)
		if err != nil {
			return err
		}
		if !sameStableIDs(rows, orderedStableIDs, func(row subLessonRow) string { return row.StableID }) {
			return domain.ErrCourseInvalidOrdering
		}
		if err := reorderStableIDRows(ctx, tx, &subLessonRow{}, "lesson_id = ?", lessonID, orderedStableIDs); err != nil {
			return err
		}
		rows, err = r.loadSubLessonsByLesson(ctx, tx, lessonID)
		if err != nil {
			return err
		}
		out = make([]domain.SubLesson, len(rows))
		for i := range rows {
			sub, err := r.loadSubLessonDomain(ctx, tx, rows[i].ID)
			if err != nil {
				return err
			}
			out[i] = *sub
		}
		return nil
	})
	return out, err
}

type outlineUpdateSpec[T any, D any] struct {
	scope              draftEditScope
	entityID           *string
	expectedRowVersion int64
	load               func(context.Context, *gorm.DB, string, string) (*T, error)
	behavior           draftEntityBehavior[T, D]
	updates            map[string]any
}

func buildOutlineUpdateSpec[T any, D any](
	scope draftEditScope,
	entityID *string,
	expectedRowVersion int64,
	load func(context.Context, *gorm.DB, string, string) (*T, error),
	behavior draftEntityBehavior[T, D],
	updates map[string]any,
) outlineUpdateSpec[T, D] {
	return outlineUpdateSpec[T, D]{
		scope:              scope,
		entityID:           entityID,
		expectedRowVersion: expectedRowVersion,
		load:               load,
		behavior:           behavior,
		updates:            updates,
	}
}

func updateSectionBySpec(
	r *GormRepository,
	ctx context.Context,
	scope draftEditScope,
	entityID *string,
	expectedRowVersion int64,
	updates map[string]any,
) (*domain.Section, error) {
	spec := buildOutlineUpdateSpec(
		scope,
		entityID,
		expectedRowVersion,
		r.loadSection,
		sectionBehavior,
		updates,
	)
	return updateSectionOrLesson(r, ctx, spec)
}

func updateLessonBySpec(
	r *GormRepository,
	ctx context.Context,
	scope draftEditScope,
	entityID *string,
	expectedRowVersion int64,
	updates map[string]any,
) (*domain.Lesson, error) {
	spec := buildOutlineUpdateSpec(
		scope,
		entityID,
		expectedRowVersion,
		r.loadLesson,
		lessonBehavior,
		updates,
	)
	return updateSectionOrLesson(r, ctx, spec)
}

func updateSectionOrLesson[T any, D any](r *GormRepository, ctx context.Context, spec outlineUpdateSpec[T, D]) (*D, error) {
	return updateOutlineEntity(
		r,
		ctx,
		spec.scope,
		spec.entityID,
		spec.expectedRowVersion,
		spec.load,
		spec.behavior,
		spec.updates,
	)
}

func (r *GormRepository) AcquireLease(ctx context.Context, courseID string, actorUserID string, in domain.AcquireLeaseInput) (*domain.Lease, error) {
	var out *domain.Lease
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := r.requireEditorAccess(ctx, tx, courseID, actorUserID); err != nil {
			return err
		}
		now := timex.NowUnix()
		var row leaseRow
		err := tx.Where("course_version_id = ? AND resource_type = ? AND resource_stable_id = ?", in.CourseVersionID, in.ResourceType, in.ResourceStableID).
			First(&row).Error
		if err == nil && row.ExpiresAt > now && row.HolderUserID != actorUserID {
			return domain.ErrCourseLeaseHeldByOtherUser
		}
		if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		expiresAt := now + 300
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			row = leaseRow{
				CourseID: courseID, CourseVersionID: in.CourseVersionID, ResourceType: in.ResourceType, ResourceStableID: in.ResourceStableID,
				HolderUserID: actorUserID, LeaseToken: uuid.NewString(), ExpiresAt: expiresAt,
			}
			if err := touchCreateCourseEntity(ctx, tx, &row.CreatedAt, &row.UpdatedAt, &row); err != nil {
				return err
			}
		} else {
			if err := tx.Model(&leaseRow{}).Where("id = ?", row.ID).Updates(map[string]any{
				"holder_user_id": actorUserID,
				"lease_token":    uuid.NewString(),
				"expires_at":     expiresAt,
				"updated_at":     timex.NowUnix(),
			}).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", row.ID).First(&row).Error; err != nil {
				return err
			}
		}
		lease := toLease(&row)
		out = &lease
		return nil
	})
	return out, err
}

func (r *GormRepository) HeartbeatLease(ctx context.Context, courseID string, actorUserID string, in domain.LeaseHeartbeatInput) (*domain.Lease, error) {
	var out *domain.Lease
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row leaseRow
		if err := tx.Where("course_id = ? AND holder_user_id = ? AND lease_token = ?", courseID, actorUserID, in.LeaseToken).First(&row).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrCourseLeaseTokenInvalid
			}
			return err
		}
		if err := tx.Model(&leaseRow{}).Where("id = ?", row.ID).Updates(map[string]any{
			"expires_at": timex.NowUnix() + 300,
			"updated_at": timex.NowUnix(),
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", row.ID).First(&row).Error; err != nil {
			return err
		}
		lease := toLease(&row)
		out = &lease
		return nil
	})
	return out, err
}

func (r *GormRepository) ReleaseLease(ctx context.Context, courseID string, actorUserID string, in domain.ReleaseLeaseInput) error {
	return r.db.WithContext(ctx).Where("course_id = ? AND holder_user_id = ? AND lease_token = ?", courseID, actorUserID, in.LeaseToken).Delete(&leaseRow{}).Error
}
