package infra

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/gormx"
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

func ensureCourseRowID(row any) error {
	switch r := row.(type) {
	case *courseRow:
		return gormx.EnsureStringID(&r.ID)
	case *courseVersionRow:
		return gormx.EnsureStringID(&r.ID)
	case *collaboratorRow:
		return gormx.EnsureStringID(&r.ID)
	case *sectionRow:
		return gormx.EnsureStringID(&r.ID)
	case *lessonRow:
		return gormx.EnsureStringID(&r.ID)
	case *subLessonRow:
		return gormx.EnsureStringID(&r.ID)
	case *enrollmentRow:
		return gormx.EnsureStringID(&r.ID)
	case *progressRow:
		return gormx.EnsureStringID(&r.ID)
	case *leaseRow:
		return gormx.EnsureStringID(&r.ID)
	case *subLessonQuizOptionRow:
		return gormx.EnsureStringID(&r.ID)
	default:
		return nil
	}
}

func touchCreateCourseEntity(ctx context.Context, tx *gorm.DB, created, updated *int64, row any) error {
	if err := ensureCourseRowID(row); err != nil {
		return err
	}
	gormx.TouchCreatedUpdated(created, updated)
	return tx.WithContext(ctx).Create(row).Error
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

const (
	maxCourseSlugLen      = 255
	courseSlugCreateRetry = 3
)

// ensureUniqueCourseSlug picks the first available slug among base, base-2, base-3, …
// among active courses. One indexed query loads sibling slugs; suffix selection is in-memory.
// excludeCourseID skips the current course on title/slug updates.
func ensureUniqueCourseSlug(ctx context.Context, db *gorm.DB, baseSlug string, excludeCourseID *string) (string, error) {
	baseSlug = strings.TrimSpace(baseSlug)
	if baseSlug == "" {
		return "", domain.ErrCourseInvalidSlug
	}
	taken, err := listCollidingCourseSlugs(ctx, db, baseSlug, excludeCourseID)
	if err != nil {
		return "", err
	}
	return nextFreeCourseSlug(baseSlug, taken), nil
}

// listCollidingCourseSlugs returns active slugs equal to base or base-<digits> only.
// LIKE base-% is a broad prefilter; non-numeric tails (e.g. base-advanced) are dropped in Go.
func listCollidingCourseSlugs(ctx context.Context, db *gorm.DB, baseSlug string, excludeCourseID *string) (map[string]struct{}, error) {
	var rows []string
	q := db.WithContext(ctx).Model(&courseRow{}).
		Where("deleted_at IS NULL").
		Where("slug = ? OR slug LIKE ?", baseSlug, baseSlug+"-%")
	if excludeCourseID != nil {
		if id := strings.TrimSpace(*excludeCourseID); id != "" {
			q = q.Where("id != ?", id)
		}
	}
	if err := q.Pluck("slug", &rows).Error; err != nil {
		return nil, err
	}
	taken := make(map[string]struct{}, len(rows))
	for _, slug := range rows {
		if isCourseSlugNumericSuffixVariant(slug, baseSlug) {
			taken[slug] = struct{}{}
		}
	}
	return taken, nil
}

func isCourseSlugNumericSuffixVariant(slug, base string) bool {
	if slug == base {
		return true
	}
	prefix := base + "-"
	if !strings.HasPrefix(slug, prefix) {
		return false
	}
	suffix := slug[len(prefix):]
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func nextFreeCourseSlug(base string, taken map[string]struct{}) string {
	if _, exists := taken[base]; !exists {
		return courseSlugCandidate(base, 1)
	}
	maxSuffix := 1
	for slug := range taken {
		if n, ok := courseSlugNumericSuffix(slug, base); ok && n > maxSuffix {
			maxSuffix = n
		}
	}
	for i := 2; i <= maxSuffix+1; i++ {
		candidate := courseSlugCandidate(base, i)
		if _, exists := taken[candidate]; !exists {
			return candidate
		}
	}
	return courseSlugCandidate(base, maxSuffix+1)
}

// courseSlugNumericSuffix returns n when slug is base-n (n >= 2).
func courseSlugNumericSuffix(slug, base string) (int, bool) {
	prefix := base + "-"
	if !strings.HasPrefix(slug, prefix) {
		return 0, false
	}
	suffix := slug[len(prefix):]
	if suffix == "" {
		return 0, false
	}
	n := 0
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	if n < 2 {
		return 0, false
	}
	return n, true
}

func courseSlugCandidate(base string, attempt int) string {
	if attempt <= 1 {
		return truncateCourseSlugBase(base, maxCourseSlugLen)
	}
	suffix := fmt.Sprintf("-%d", attempt)
	maxBase := maxCourseSlugLen - len(suffix)
	return truncateCourseSlugBase(base, maxBase) + suffix
}

func truncateCourseSlugBase(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return strings.TrimRight(s, "-")
	}
	cut := s[:maxBytes]
	for !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimRight(cut, "-")
}

func isCourseSlugDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") &&
		(strings.Contains(msg, "uix_courses_slug_active") || strings.Contains(msg, "courses_slug"))
}

// reorderStableIDRows assigns order_index for active rows scoped by scopeQuery/scopeValue.
// It uses a two-phase update (temporary negative indices, then final 0..n-1) so partial
// reorders cannot violate unique (scope, order_index) indexes mid-transaction.
func reorderStableIDRows(
	ctx context.Context,
	tx *gorm.DB,
	model any,
	scopeQuery string,
	scopeValue string,
	orderedStableIDs []string,
) error {
	now := timex.NowUnix()
	apply := func(orderIndex int, stableID string) error {
		return tx.WithContext(ctx).Model(model).
			Where(scopeQuery+" AND stable_id = ? AND deleted_at IS NULL", scopeValue, stableID).
			Updates(map[string]any{
				"order_index": orderIndex,
				"updated_at":  now,
				"row_version": gorm.Expr("row_version + 1"),
			}).Error
	}
	for idx, stableID := range orderedStableIDs {
		if err := apply(-(idx+1), stableID); err != nil {
			return err
		}
	}
	for idx, stableID := range orderedStableIDs {
		if err := apply(idx, stableID); err != nil {
			return err
		}
	}
	return nil
}
