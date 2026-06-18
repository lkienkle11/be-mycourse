package infra

import (
	"context"
	stderrors "errors"
	"sort"
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
	var versionID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		versionID = *access.CurrentDraftVersionID
		rows, err := r.loadSectionsByVersion(ctx, tx, versionID)
		if err != nil {
			return err
		}
		if !sameStableIDs(rows, orderedStableIDs, func(row sectionRow) string { return row.StableID }) {
			return domain.ErrCourseInvalidOrdering
		}
		return reorderStableIDRows(ctx, tx, &sectionRow{}, "course_version_id = ?", versionID, orderedStableIDs)
	})
	if err != nil {
		return nil, err
	}
	return r.loadOutline(ctx, r.db, versionID)
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
		out, err = r.hydrateLessonRowsWithSubLessons(ctx, tx, rows, orderedStableIDs)
		return err
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
		estimatedMs, err := normalizeSubLessonEstimatedDurationMs(in.Kind, in.EstimatedDurationMs)
		if err != nil {
			return err
		}
		next, err := r.nextOrderIndex(ctx, tx, constants.TableCourseSubLessons, "lesson_id = ? AND deleted_at IS NULL", in.LessonID)
		if err != nil {
			return err
		}
		row := &subLessonRow{
			StableID: uuid.NewString(), CourseVersionID: *access.CurrentDraftVersionID, LessonID: in.LessonID,
			Title: strings.TrimSpace(in.Title), Kind: strings.ToUpper(strings.TrimSpace(in.Kind)), IsPreview: in.IsPreview,
			EstimatedDurationMs: estimatedMs, OrderIndex: next, RowVersion: 1,
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
		if err := r.resolveSubLessonDomainDuration(ctx, tx, sub); err != nil {
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
		estimatedMs, err := normalizeSubLessonEstimatedDurationMs(in.Kind, in.EstimatedDurationMs)
		if err != nil {
			return err
		}
		result := tx.Model(&subLessonRow{}).
			Where("id = ? AND row_version = ? AND deleted_at IS NULL", row.ID, in.ExpectedRowVersion).
			Updates(map[string]any{
				"title":                 strings.TrimSpace(in.Title),
				"kind":                  strings.ToUpper(strings.TrimSpace(in.Kind)),
				"is_preview":            in.IsPreview,
				"estimated_duration_ms": estimatedMs,
				"updated_at":            timex.NowUnix(),
				"row_version":           gorm.Expr("row_version + 1"),
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
		if err := r.resolveSubLessonDomainDuration(ctx, tx, sub); err != nil {
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
		outline, err = r.loadOutlineSequential(ctx, tx, *access.CurrentDraftVersionID)
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
		applyStableIDReorderRowMeta(rows, orderedStableIDs)
		videoMediaMs := make(map[string]int64)
		subMap, err := r.batchHydrateSubLessons(ctx, tx, rows, videoMediaMs)
		if err != nil {
			return err
		}
		out = make([]domain.SubLesson, len(rows))
		for i, row := range rows {
			sub, ok := subMap[row.ID]
			if !ok {
				return domain.ErrCourseNotFound
			}
			out[i] = sub
		}
		applySubLessonListEstimatedDurations(out, videoMediaMs)
		return nil
	})
	return out, err
}

func (r *GormRepository) hydrateLessonRowsWithSubLessons(
	ctx context.Context,
	db *gorm.DB,
	lessonRows []lessonRow,
	orderedStableIDs []string,
) ([]domain.Lesson, error) {
	if len(lessonRows) == 0 {
		return []domain.Lesson{}, nil
	}
	orderByStable := make(map[string]int, len(orderedStableIDs))
	for i, stableID := range orderedStableIDs {
		orderByStable[stableID] = i
	}
	sort.Slice(lessonRows, func(i, j int) bool {
		return orderByStable[lessonRows[i].StableID] < orderByStable[lessonRows[j].StableID]
	})
	lessonIDs := make([]string, len(lessonRows))
	for i, row := range lessonRows {
		lessonIDs[i] = row.ID
	}
	subRows, err := loadActiveRows[subLessonRow](ctx, db, "lesson_id IN ? AND deleted_at IS NULL", lessonIDs)
	if err != nil {
		return nil, err
	}
	videoMediaMs := make(map[string]int64)
	subMap, err := r.batchHydrateSubLessons(ctx, db, subRows, videoMediaMs)
	if err != nil {
		return nil, err
	}
	subLessonsByLesson := make(map[string][]domain.SubLesson, len(lessonRows))
	for _, sub := range subRows {
		hydrated, ok := subMap[sub.ID]
		if !ok {
			return nil, domain.ErrCourseNotFound
		}
		subLessonsByLesson[sub.LessonID] = append(subLessonsByLesson[sub.LessonID], hydrated)
	}
	out := make([]domain.Lesson, len(lessonRows))
	for i, row := range lessonRows {
		out[i] = toLesson(&row)
		out[i].SubLessons = subLessonsByLesson[row.ID]
		if out[i].SubLessons == nil {
			out[i].SubLessons = []domain.SubLesson{}
		}
	}
	videoIDs := collectVideoMediaFileIDs([]domain.Section{{Lessons: out}})
	if len(videoIDs) > 0 {
		loaded, err := r.batchMediaDurationMs(ctx, db, videoIDs)
		if err != nil {
			return nil, err
		}
		for id, ms := range loaded {
			videoMediaMs[id] = ms
		}
	}
	applyOutlineEstimatedDurations([]domain.Section{{Lessons: out}}, videoMediaMs)
	return out, nil
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
			newToken := uuid.NewString()
			if err := tx.Model(&leaseRow{}).Where("id = ?", row.ID).Updates(map[string]any{
				"holder_user_id": actorUserID,
				"lease_token":    newToken,
				"expires_at":     expiresAt,
				"updated_at":     now,
			}).Error; err != nil {
				return err
			}
			row.HolderUserID = actorUserID
			row.LeaseToken = newToken
			row.ExpiresAt = expiresAt
			row.UpdatedAt = now
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
		now := timex.NowUnix()
		expiresAt := now + 300
		if err := tx.Model(&leaseRow{}).Where("id = ?", row.ID).Updates(map[string]any{
			"expires_at": expiresAt,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		row.ExpiresAt = expiresAt
		row.UpdatedAt = now
		lease := toLease(&row)
		out = &lease
		return nil
	})
	return out, err
}

func (r *GormRepository) ReleaseLease(ctx context.Context, courseID string, actorUserID string, in domain.ReleaseLeaseInput) error {
	return r.db.WithContext(ctx).Where("course_id = ? AND holder_user_id = ? AND lease_token = ?", courseID, actorUserID, in.LeaseToken).Delete(&leaseRow{}).Error
}
