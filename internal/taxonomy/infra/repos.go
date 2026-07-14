// Package infra contains the TAXONOMY bounded-context infrastructure (GORM repositories).
package infra

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/gormx"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	sharedutils "mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/taxonomy/domain"
)

// --- GORM row types ----------------------------------------------------------

type courseTopicRow struct {
	ID          string         `gorm:"column:id;primaryKey;type:uuid"`
	Name        string         `gorm:"size:255;not null"`
	Slug        string         `gorm:"size:255;not null;index"`
	ImageFileID *string        `gorm:"column:image_file_id;type:uuid"`
	ChildTopics treeNodesJSONB `gorm:"column:child_topics;type:jsonb;not null;default:'[]'"`
	Status      string         `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	RowVersion  int64          `gorm:"column:row_version;not null;default:1"`
	CreatedBy   *string        `gorm:"column:created_by;type:uuid"`
	CreatedAt   int64
	UpdatedAt   int64
	DeletedAt   *int64 `gorm:"column:deleted_at;index"`
}

func (courseTopicRow) TableName() string { return constants.TableTaxonomyCourseTopics }

type courseOutcomeRow struct {
	ID               string           `gorm:"column:id;primaryKey;type:uuid"`
	ShortDescription string           `gorm:"column:short_description;size:100;not null"`
	Description      descriptionJSONB `gorm:"type:jsonb;not null;default:'[]'"`
	ImageFileID      *string          `gorm:"column:image_file_id;type:uuid"`
	Status           string           `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	RowVersion       int64            `gorm:"column:row_version;not null;default:1"`
	CreatedBy        *string          `gorm:"column:created_by;type:uuid"`
	CreatedAt        int64
	UpdatedAt        int64
	DeletedAt        *int64 `gorm:"column:deleted_at;index"`
}

func (courseOutcomeRow) TableName() string { return constants.TableTaxonomyCourseOutcomes }

type courseSkillRow struct {
	ID         string         `gorm:"column:id;primaryKey;type:uuid"`
	Name       string         `gorm:"size:255;not null"`
	Slug       string         `gorm:"size:255;not null;index"`
	Children   treeNodesJSONB `gorm:"type:jsonb;not null;default:'[]'"`
	Status     string         `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	RowVersion int64          `gorm:"column:row_version;not null;default:1"`
	CreatedBy  *string        `gorm:"column:created_by;type:uuid"`
	CreatedAt  int64
	UpdatedAt  int64
	DeletedAt  *int64 `gorm:"column:deleted_at;index"`
}

func (courseSkillRow) TableName() string { return constants.TableTaxonomyCourseSkills }

type tagRow struct {
	ID         string  `gorm:"column:id;primaryKey;type:uuid"`
	Name       string  `gorm:"size:255;not null"`
	Slug       string  `gorm:"size:255;not null;index"`
	Status     string  `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	RowVersion int64   `gorm:"column:row_version;not null;default:1"`
	CreatedBy  *string `gorm:"column:created_by;type:uuid"`
	CreatedAt  int64
	UpdatedAt  int64
	DeletedAt  *int64 `gorm:"column:deleted_at;index"`
}

func (tagRow) TableName() string { return constants.TableTaxonomyTags }

type courseLevelRow struct {
	ID         string  `gorm:"column:id;primaryKey;type:uuid"`
	Name       string  `gorm:"size:255;not null"`
	Slug       string  `gorm:"size:255;not null;index"`
	Status     string  `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	RowVersion int64   `gorm:"column:row_version;not null;default:1"`
	CreatedBy  *string `gorm:"column:created_by;type:uuid"`
	CreatedAt  int64
	UpdatedAt  int64
	DeletedAt  *int64 `gorm:"column:deleted_at;index"`
}

func (courseLevelRow) TableName() string { return constants.TableTaxonomyCourseLevels }

// --- query helpers -----------------------------------------------------------

var taxonomySearchCols = map[string]string{"name": "name", "slug": "slug"}
var outcomeSearchCols = map[string]string{"short_description": "short_description"}
var sortCols = map[string]string{
	"id": "id", "name": "name", "slug": "slug", "status": "status", "created_at": "created_at",
}

func applyTaxonomyFilter(q *gorm.DB, filter domain.TaxonomyFilter) *gorm.DB {
	q = applyStatusFilter(q, filter.Status)
	return applySearchByFilter(q, filter, taxonomySearchCols)
}

func applyOutcomeSearch(q *gorm.DB, filter domain.TaxonomyFilter) *gorm.DB {
	q = applyStatusFilter(q, filter.Status)
	return applySearchByFilter(q, filter, outcomeSearchCols)
}

func applyStatusFilter(q *gorm.DB, status *string) *gorm.DB {
	if status == nil || strings.TrimSpace(*status) == "" {
		return q
	}
	return q.Where("status = ?", strings.ToUpper(strings.TrimSpace(*status)))
}

func applySearchByFilter(q *gorm.DB, filter domain.TaxonomyFilter, allowed map[string]string) *gorm.DB {
	base := sharedutils.BaseFilter{
		SearchBy:   strings.ToLower(strings.TrimSpace(filter.SearchBy)),
		SearchData: strings.TrimSpace(filter.SearchValue),
	}
	clause, value, ok := sharedutils.BuildSearchClause(base, allowed)
	if !ok {
		return q
	}
	return q.Where(clause, value)
}

func applyPagination(q *gorm.DB, filter domain.TaxonomyFilter) *gorm.DB {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	sortBy, ok := sortCols[filter.SortBy]
	if !ok {
		sortBy = "id"
	}
	order := sortBy + " ASC"
	if filter.SortDesc {
		order = sortBy + " DESC"
	}
	return q.Order(order).Offset((page - 1) * pageSize).Limit(pageSize)
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrNotFound
	}
	return err
}

// --- GormCourseTopicRepository -----------------------------------------------

// GormCourseTopicRepository implements domain.CourseTopicRepository.
type GormCourseTopicRepository struct{ db *gorm.DB }

// NewGormCourseTopicRepository constructs a GormCourseTopicRepository.
func NewGormCourseTopicRepository(db *gorm.DB) *GormCourseTopicRepository {
	return &GormCourseTopicRepository{db: db}
}

func mapCourseTopicRow(row *courseTopicRow) domain.CourseTopic {
	return rowToCourseTopic(row, nil)
}

func (r *GormCourseTopicRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseTopic, int64, error) {
	return listHydratedWithImages(ctx, r.db, listHydratedWithImagesArgs[courseTopicRow, domain.CourseTopic]{
		Model: &courseTopicRow{}, Filter: filter, ApplySearch: applyTaxonomyFilter,
		MapRow: mapCourseTopicRow, GetID: topicImageID, SetImage: applyTopicImageRow,
		Hydrate: hydrateCourseTopicsLocale,
	})
}

func (r *GormCourseTopicRepository) GetByID(ctx context.Context, id string) (*domain.CourseTopic, error) {
	return taxonomyGetByIDWithImageURLs(ctx, r.db, id, mapCourseTopicRow, topicImageID, applyTopicImageRow)
}

func (r *GormCourseTopicRepository) Create(ctx context.Context, t *domain.CourseTopic) error {
	return createWithNameTranslations(ctx, r.db, createNameTranslationArgs[courseTopicRow, domain.CourseTopic]{
		Entity: t, RowVersion: &t.RowVersion, ToRow: courseTopicToRow,
		EntityID: &t.ID, CreatedAt: &t.CreatedAt, UpdatedAt: &t.UpdatedAt, Meta: metaCourseTopicRow,
		Table: constants.TableTaxonomyCourseTopicTranslations, FKCol: "topic_id", Translations: t.Translations,
	})
}

func (r *GormCourseTopicRepository) Save(ctx context.Context, t *domain.CourseTopic, expectedRowVersion int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		child := t.ChildTopics
		if child == nil {
			child = []taxpkg.TreeNode{}
		}
		childVal, err := treeNodesJSONB(child).Value()
		if err != nil {
			return err
		}
		updates := map[string]any{
			"name": t.Name, "slug": t.Slug, "status": t.Status,
			"image_file_id": t.ImageFileID, "child_topics": childVal,
		}
		if err := gormx.OptimisticUpdate(ctx, tx, &courseTopicRow{}, t.ID, expectedRowVersion, updates, domain.ErrTaxonomyOptimisticLock); err != nil {
			return err
		}
		if err := upsertNameTranslations(ctx, tx, constants.TableTaxonomyCourseTopicTranslations, "topic_id", t.ID, t.Translations); err != nil {
			return err
		}
		t.RowVersion = expectedRowVersion + 1
		if v, ok := updates["updated_at"].(int64); ok {
			t.UpdatedAt = v
		}
		return nil
	})
}

func (r *GormCourseTopicRepository) SoftDelete(ctx context.Context, id string) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseTopicRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseTopicRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&courseTopicRow{}, id).Error
}

// --- GormCourseOutcomeRepository ---------------------------------------------

// GormCourseOutcomeRepository implements domain.CourseOutcomeRepository.
type GormCourseOutcomeRepository struct{ db *gorm.DB }

// NewGormCourseOutcomeRepository constructs a GormCourseOutcomeRepository.
func NewGormCourseOutcomeRepository(db *gorm.DB) *GormCourseOutcomeRepository {
	return &GormCourseOutcomeRepository{db: db}
}

func mapCourseOutcomeRow(row *courseOutcomeRow) domain.CourseOutcome {
	return rowToCourseOutcome(row, nil)
}

func (r *GormCourseOutcomeRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseOutcome, int64, error) {
	args := listHydratedWithImagesArgs[courseOutcomeRow, domain.CourseOutcome]{
		Model: &courseOutcomeRow{}, Filter: filter, ApplySearch: applyOutcomeSearch,
		MapRow: mapCourseOutcomeRow, GetID: outcomeImageID, SetImage: applyOutcomeImageRow,
		Hydrate: hydrateCourseOutcomesLocale,
	}
	return listHydratedWithImages(ctx, r.db, args)
}

func (r *GormCourseOutcomeRepository) GetByID(ctx context.Context, id string) (*domain.CourseOutcome, error) {
	return taxonomyGetByIDWithImageURLs(ctx, r.db, id, mapCourseOutcomeRow, outcomeImageID, applyOutcomeImageRow)
}

func (r *GormCourseOutcomeRepository) Create(ctx context.Context, o *domain.CourseOutcome) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if o.RowVersion == 0 {
			o.RowVersion = 1
		}
		if err := createTaxonomyDomain(ctx, tx, o, courseOutcomeToRow, &o.ID, &o.CreatedAt, &o.UpdatedAt, metaCourseOutcomeRow); err != nil {
			return err
		}
		return upsertOutcomeTranslations(ctx, tx, o.ID, o.Translations)
	})
}

func (r *GormCourseOutcomeRepository) Save(ctx context.Context, o *domain.CourseOutcome, expectedRowVersion int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		desc := o.Description
		if desc == nil {
			desc = []string{}
		}
		descVal, err := descriptionJSONB(desc).Value()
		if err != nil {
			return err
		}
		updates := map[string]any{
			"short_description": o.ShortDescription,
			"description":       descVal,
			"status":            o.Status,
			"image_file_id":     o.ImageFileID,
		}
		if err := gormx.OptimisticUpdate(ctx, tx, &courseOutcomeRow{}, o.ID, expectedRowVersion, updates, domain.ErrTaxonomyOptimisticLock); err != nil {
			return err
		}
		if err := upsertOutcomeTranslations(ctx, tx, o.ID, o.Translations); err != nil {
			return err
		}
		o.RowVersion = expectedRowVersion + 1
		if v, ok := updates["updated_at"].(int64); ok {
			o.UpdatedAt = v
		}
		return nil
	})
}

func (r *GormCourseOutcomeRepository) SoftDelete(ctx context.Context, id string) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseOutcomeRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseOutcomeRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&courseOutcomeRow{}, id).Error
}

// --- GormCourseSkillRepository -----------------------------------------------

// GormCourseSkillRepository implements domain.CourseSkillRepository.
type GormCourseSkillRepository struct{ db *gorm.DB }

// NewGormCourseSkillRepository constructs a GormCourseSkillRepository.
func NewGormCourseSkillRepository(db *gorm.DB) *GormCourseSkillRepository {
	return &GormCourseSkillRepository{db: db}
}

func (r *GormCourseSkillRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseSkill, int64, error) {
	return listHydrated(ctx, r.db, &courseSkillRow{}, filter, applyTaxonomyFilter, rowToCourseSkill, hydrateCourseSkillsLocale)
}

func (r *GormCourseSkillRepository) GetByID(ctx context.Context, id string) (*domain.CourseSkill, error) {
	return taxonomyGetByID(ctx, r.db, id, rowToCourseSkill)
}

func (r *GormCourseSkillRepository) Create(ctx context.Context, s *domain.CourseSkill) error {
	args := createNameTranslationArgs[courseSkillRow, domain.CourseSkill]{
		Entity: s, RowVersion: &s.RowVersion, ToRow: courseSkillToRow,
		EntityID: &s.ID, CreatedAt: &s.CreatedAt, UpdatedAt: &s.UpdatedAt, Meta: metaCourseSkillRow,
		Table: constants.TableTaxonomyCourseSkillTranslations, FKCol: "skill_id", Translations: s.Translations,
	}
	return createWithNameTranslations(ctx, r.db, args)
}

func (r *GormCourseSkillRepository) Save(ctx context.Context, s *domain.CourseSkill, expectedRowVersion int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		child := s.Children
		if child == nil {
			child = []taxpkg.TreeNode{}
		}
		childVal, err := treeNodesJSONB(child).Value()
		if err != nil {
			return err
		}
		updates := map[string]any{
			"name": s.Name, "slug": s.Slug, "status": s.Status,
			"children": childVal,
		}
		if err := gormx.OptimisticUpdate(ctx, tx, &courseSkillRow{}, s.ID, expectedRowVersion, updates, domain.ErrTaxonomyOptimisticLock); err != nil {
			return err
		}
		if err := upsertNameTranslations(ctx, tx, constants.TableTaxonomyCourseSkillTranslations, "skill_id", s.ID, s.Translations); err != nil {
			return err
		}
		s.RowVersion = expectedRowVersion + 1
		if v, ok := updates["updated_at"].(int64); ok {
			s.UpdatedAt = v
		}
		return nil
	})
}

func (r *GormCourseSkillRepository) SoftDelete(ctx context.Context, id string) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseSkillRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseSkillRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&courseSkillRow{}, id).Error
}

// --- GormTagRepository -------------------------------------------------------

// GormTagRepository implements domain.TagRepository.
type GormTagRepository struct{ db *gorm.DB }

// NewGormTagRepository constructs a GormTagRepository.
func NewGormTagRepository(db *gorm.DB) *GormTagRepository {
	return &GormTagRepository{db: db}
}

func (r *GormTagRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.Tag, int64, error) {
	return listHydrated(ctx, r.db, &tagRow{}, filter, applyTaxonomyFilter, rowToTag, hydrateTagsLocale)
}

func (r *GormTagRepository) GetByID(ctx context.Context, id string) (*domain.Tag, error) {
	return taxonomyGetByID(ctx, r.db, id, rowToTag)
}

func (r *GormTagRepository) Create(ctx context.Context, t *domain.Tag) error {
	args := createNameTranslationArgs[tagRow, domain.Tag]{Entity: t, RowVersion: &t.RowVersion}
	args.ToRow = tagToRow
	args.EntityID = &t.ID
	args.CreatedAt = &t.CreatedAt
	args.UpdatedAt = &t.UpdatedAt
	args.Meta = metaTagRow
	args.Table = constants.TableTaxonomyTagTranslations
	args.FKCol = "tag_id"
	args.Translations = t.Translations
	return createWithNameTranslations(ctx, r.db, args)
}

func (r *GormTagRepository) Save(ctx context.Context, t *domain.Tag, expectedRowVersion int64) error {
	return saveSlugStatusEntity(ctx, r.db, slugStatusSaveArgs[tagRow]{
		Model: &tagRow{}, ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
		ExpectedRowVersion: expectedRowVersion, RowVersion: &t.RowVersion, UpdatedAt: &t.UpdatedAt,
		Table: constants.TableTaxonomyTagTranslations, FKCol: "tag_id", Translations: t.Translations,
	})
}

func (r *GormTagRepository) SoftDelete(ctx context.Context, id string) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &tagRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormTagRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&tagRow{}, id).Error
}

// --- GormCourseLevelRepository -----------------------------------------------

// GormCourseLevelRepository implements domain.CourseLevelRepository.
type GormCourseLevelRepository struct{ db *gorm.DB }

// NewGormCourseLevelRepository constructs a GormCourseLevelRepository.
func NewGormCourseLevelRepository(db *gorm.DB) *GormCourseLevelRepository {
	return &GormCourseLevelRepository{db: db}
}

func (r *GormCourseLevelRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseLevel, int64, error) {
	return listHydrated(ctx, r.db, &courseLevelRow{}, filter, applyTaxonomyFilter, rowToCourseLevel, hydrateCourseLevelsLocale)
}

func (r *GormCourseLevelRepository) GetByID(ctx context.Context, id string) (*domain.CourseLevel, error) {
	return taxonomyGetByID(ctx, r.db, id, rowToCourseLevel)
}

func (r *GormCourseLevelRepository) Create(ctx context.Context, cl *domain.CourseLevel) error {
	args := createNameTranslationArgs[courseLevelRow, domain.CourseLevel]{Entity: cl}
	args.RowVersion = &cl.RowVersion
	args.ToRow = courseLevelToRow
	args.EntityID, args.CreatedAt, args.UpdatedAt = &cl.ID, &cl.CreatedAt, &cl.UpdatedAt
	args.Meta = metaCourseLevelRow
	args.Table, args.FKCol = constants.TableTaxonomyCourseLevelTranslations, "course_level_id"
	args.Translations = cl.Translations
	return createWithNameTranslations(ctx, r.db, args)
}

func (r *GormCourseLevelRepository) Save(ctx context.Context, cl *domain.CourseLevel, expectedRowVersion int64) error {
	args := slugStatusSaveArgs[courseLevelRow]{
		Model: &courseLevelRow{}, ID: cl.ID, Name: cl.Name, Slug: cl.Slug, Status: cl.Status,
		ExpectedRowVersion: expectedRowVersion, RowVersion: &cl.RowVersion, UpdatedAt: &cl.UpdatedAt,
		Table: constants.TableTaxonomyCourseLevelTranslations, FKCol: "course_level_id", Translations: cl.Translations,
	}
	return saveSlugStatusEntity(ctx, r.db, args)
}

func (r *GormCourseLevelRepository) SoftDelete(ctx context.Context, id string) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseLevelRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseLevelRepository) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&courseLevelRow{}, id).Error
}

type slugStatusSaveArgs[R any] struct {
	Model              *R
	ID                 string
	Name               string
	Slug               string
	Status             string
	ExpectedRowVersion int64
	RowVersion         *int64
	UpdatedAt          *int64
	Table              string
	FKCol              string
	Translations       map[string]taxpkg.NodeTranslation
}

func saveSlugStatusEntity[R any](ctx context.Context, db *gorm.DB, args slugStatusSaveArgs[R]) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"name": args.Name, "slug": args.Slug, "status": args.Status}
		if err := gormx.OptimisticUpdate(ctx, tx, args.Model, args.ID, args.ExpectedRowVersion, updates, domain.ErrTaxonomyOptimisticLock); err != nil {
			return err
		}
		if err := upsertNameTranslations(ctx, tx, args.Table, args.FKCol, args.ID, args.Translations); err != nil {
			return err
		}
		*args.RowVersion = args.ExpectedRowVersion + 1
		if v, ok := updates["updated_at"].(int64); ok {
			*args.UpdatedAt = v
		}
		return nil
	})
}
