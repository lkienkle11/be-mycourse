// Package infra contains the TAXONOMY bounded-context infrastructure (GORM repositories).
package infra

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/taxonomy/domain"
)

// --- GORM row types ----------------------------------------------------------

type categoryRow struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"`
	Name        string  `gorm:"size:255;not null"`
	Slug        string  `gorm:"size:255;not null;uniqueIndex"`
	ImageFileID *string `gorm:"column:image_file_id;type:uuid"`
	Status      string  `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy   *uint   `gorm:"column:created_by"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (categoryRow) TableName() string { return constants.TableTaxonomyCategories }

type tagRow struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"size:255;not null"`
	Slug      string `gorm:"size:255;not null;uniqueIndex"`
	Status    string `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy *uint  `gorm:"column:created_by"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (tagRow) TableName() string { return constants.TableTaxonomyTags }

type courseLevelRow struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"size:255;not null"`
	Slug      string `gorm:"size:255;not null;uniqueIndex"`
	Status    string `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy *uint  `gorm:"column:created_by"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (courseLevelRow) TableName() string { return constants.TableTaxonomyCourseLevels }

// --- query helpers -----------------------------------------------------------

var searchCols = map[string]string{"name": "name", "slug": "slug"}
var sortCols = map[string]string{
	"id": "id", "name": "name", "slug": "slug", "status": "status", "created_at": "created_at",
}

func applyTaxonomyFilter(q *gorm.DB, filter domain.TaxonomyFilter) *gorm.DB {
	if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
		q = q.Where("status = ?", strings.ToUpper(strings.TrimSpace(*filter.Status)))
	}
	if filter.Search != "" {
		col, ok := searchCols["name"]
		if ok {
			q = q.Where(col+" ILIKE ?", "%"+filter.Search+"%")
		}
	}
	return q
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

// --- GormCategoryRepository --------------------------------------------------

// GormCategoryRepository implements domain.CategoryRepository.
type GormCategoryRepository struct{ db *gorm.DB }

func NewGormCategoryRepository(db *gorm.DB) *GormCategoryRepository {
	return &GormCategoryRepository{db: db}
}

func (r *GormCategoryRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.Category, int64, error) {
	q := applyTaxonomyFilter(r.db.WithContext(ctx).Model(&categoryRow{}), filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []categoryRow
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Category, len(rows))
	for i := range rows {
		out[i] = rowToCategory(&rows[i], nil)
	}
	return out, total, nil
}

func (r *GormCategoryRepository) GetByID(ctx context.Context, id uint) (*domain.Category, error) {
	var row categoryRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	cat := rowToCategory(&row, nil)
	return &cat, nil
}

func (r *GormCategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	row := categoryToRow(c)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	c.ID = row.ID
	c.CreatedAt = row.CreatedAt
	c.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormCategoryRepository) Save(ctx context.Context, c *domain.Category) error {
	row := categoryToRow(c)
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormCategoryRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&categoryRow{}, id).Error
}

// --- GormTagRepository -------------------------------------------------------

// GormTagRepository implements domain.TagRepository.
type GormTagRepository struct{ db *gorm.DB }

func NewGormTagRepository(db *gorm.DB) *GormTagRepository {
	return &GormTagRepository{db: db}
}

func (r *GormTagRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.Tag, int64, error) {
	q := applyTaxonomyFilter(r.db.WithContext(ctx).Model(&tagRow{}), filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []tagRow
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Tag, len(rows))
	for i := range rows {
		out[i] = rowToTag(&rows[i])
	}
	return out, total, nil
}

func (r *GormTagRepository) GetByID(ctx context.Context, id uint) (*domain.Tag, error) {
	var row tagRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	t := rowToTag(&row)
	return &t, nil
}

func (r *GormTagRepository) Create(ctx context.Context, t *domain.Tag) error {
	row := tagToRow(t)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	t.ID = row.ID
	t.CreatedAt = row.CreatedAt
	t.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormTagRepository) Save(ctx context.Context, t *domain.Tag) error {
	return r.db.WithContext(ctx).Save(tagToRow(t)).Error
}

func (r *GormTagRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&tagRow{}, id).Error
}

// --- GormCourseLevelRepository -----------------------------------------------

// GormCourseLevelRepository implements domain.CourseLevelRepository.
type GormCourseLevelRepository struct{ db *gorm.DB }

func NewGormCourseLevelRepository(db *gorm.DB) *GormCourseLevelRepository {
	return &GormCourseLevelRepository{db: db}
}

func (r *GormCourseLevelRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseLevel, int64, error) {
	q := applyTaxonomyFilter(r.db.WithContext(ctx).Model(&courseLevelRow{}), filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []courseLevelRow
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CourseLevel, len(rows))
	for i := range rows {
		out[i] = rowToCourseLevel(&rows[i])
	}
	return out, total, nil
}

func (r *GormCourseLevelRepository) GetByID(ctx context.Context, id uint) (*domain.CourseLevel, error) {
	var row courseLevelRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	cl := rowToCourseLevel(&row)
	return &cl, nil
}

func (r *GormCourseLevelRepository) Create(ctx context.Context, cl *domain.CourseLevel) error {
	row := courseLevelToRow(cl)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	cl.ID = row.ID
	cl.CreatedAt = row.CreatedAt
	cl.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormCourseLevelRepository) Save(ctx context.Context, cl *domain.CourseLevel) error {
	return r.db.WithContext(ctx).Save(courseLevelToRow(cl)).Error
}

func (r *GormCourseLevelRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&courseLevelRow{}, id).Error
}

// --- row mappers -------------------------------------------------------------

type imageFileRow struct {
	URL  string
	Kind string
	Mime string
}

func rowToCategory(r *categoryRow, img *imageFileRow) domain.Category {
	c := domain.Category{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		ImageFileID: r.ImageFileID, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if img != nil {
		c.ImageFileURL = img.URL
		c.ImageFileKind = img.Kind
		c.ImageFileMime = img.Mime
	}
	return c
}

func categoryToRow(c *domain.Category) *categoryRow {
	return &categoryRow{
		ID: c.ID, Name: c.Name, Slug: c.Slug, Status: c.Status,
		ImageFileID: c.ImageFileID, CreatedBy: c.CreatedBy,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func rowToTag(r *tagRow) domain.Tag {
	return domain.Tag{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func tagToRow(t *domain.Tag) *tagRow {
	return &tagRow{
		ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
		CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func rowToCourseLevel(r *courseLevelRow) domain.CourseLevel {
	return domain.CourseLevel{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func courseLevelToRow(cl *domain.CourseLevel) *courseLevelRow {
	return &courseLevelRow{
		ID: cl.ID, Name: cl.Name, Slug: cl.Slug, Status: cl.Status,
		CreatedBy: cl.CreatedBy, CreatedAt: cl.CreatedAt, UpdatedAt: cl.UpdatedAt,
	}
}
