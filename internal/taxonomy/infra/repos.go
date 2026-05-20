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
	taxpkg "mycourse-io-be/pkg/taxonomy"
)

// --- GORM row types ----------------------------------------------------------

type courseTopicRow struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	Name        string         `gorm:"size:255;not null"`
	Slug        string         `gorm:"size:255;not null;uniqueIndex"`
	ImageFileID *string        `gorm:"column:image_file_id;type:uuid"`
	ChildTopics treeNodesJSONB `gorm:"column:child_topics;type:jsonb;not null;default:'[]'"`
	Status      string         `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy   *uint          `gorm:"column:created_by"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (courseTopicRow) TableName() string { return constants.TableTaxonomyCourseTopics }

type courseOutcomeRow struct {
	ID               uint             `gorm:"primaryKey;autoIncrement"`
	ShortDescription string           `gorm:"column:short_description;size:100;not null"`
	Description      descriptionJSONB `gorm:"type:jsonb;not null;default:'[]'"`
	ImageFileID      *string          `gorm:"column:image_file_id;type:uuid"`
	Status           string           `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy        *uint            `gorm:"column:created_by"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (courseOutcomeRow) TableName() string { return constants.TableTaxonomyCourseOutcomes }

type courseSkillRow struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	Name      string         `gorm:"size:255;not null"`
	Slug      string         `gorm:"size:255;not null;uniqueIndex"`
	Children  treeNodesJSONB `gorm:"type:jsonb;not null;default:'[]'"`
	Status    string         `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy *uint          `gorm:"column:created_by"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (courseSkillRow) TableName() string { return constants.TableTaxonomyCourseSkills }

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

func applyOutcomeSearch(q *gorm.DB, filter domain.TaxonomyFilter) *gorm.DB {
	q = applyTaxonomyFilter(q, filter)
	if filter.Search != "" {
		q = q.Where("short_description ILIKE ?", "%"+filter.Search+"%")
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

// --- GormCourseTopicRepository -----------------------------------------------

// GormCourseTopicRepository implements domain.CourseTopicRepository.
type GormCourseTopicRepository struct{ db *gorm.DB }

// NewGormCourseTopicRepository constructs a GormCourseTopicRepository.
func NewGormCourseTopicRepository(db *gorm.DB) *GormCourseTopicRepository {
	return &GormCourseTopicRepository{db: db}
}

func (r *GormCourseTopicRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseTopic, int64, error) {
	q := applyTaxonomyFilter(r.db.WithContext(ctx).Model(&courseTopicRow{}), filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []courseTopicRow
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CourseTopic, len(rows))
	for i := range rows {
		out[i] = rowToCourseTopic(&rows[i], nil)
	}
	return out, total, nil
}

func (r *GormCourseTopicRepository) GetByID(ctx context.Context, id uint) (*domain.CourseTopic, error) {
	var row courseTopicRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	t := rowToCourseTopic(&row, nil)
	return &t, nil
}

func (r *GormCourseTopicRepository) Create(ctx context.Context, t *domain.CourseTopic) error {
	row := courseTopicToRow(t)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	t.ID = row.ID
	t.CreatedAt = row.CreatedAt
	t.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormCourseTopicRepository) Save(ctx context.Context, t *domain.CourseTopic) error {
	return r.db.WithContext(ctx).Save(courseTopicToRow(t)).Error
}

func (r *GormCourseTopicRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&courseTopicRow{}, id).Error
}

// --- GormCourseOutcomeRepository ---------------------------------------------

// GormCourseOutcomeRepository implements domain.CourseOutcomeRepository.
type GormCourseOutcomeRepository struct{ db *gorm.DB }

// NewGormCourseOutcomeRepository constructs a GormCourseOutcomeRepository.
func NewGormCourseOutcomeRepository(db *gorm.DB) *GormCourseOutcomeRepository {
	return &GormCourseOutcomeRepository{db: db}
}

func (r *GormCourseOutcomeRepository) List(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseOutcome, int64, error) {
	q := applyOutcomeSearch(r.db.WithContext(ctx).Model(&courseOutcomeRow{}), filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []courseOutcomeRow
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CourseOutcome, len(rows))
	for i := range rows {
		out[i] = rowToCourseOutcome(&rows[i], nil)
	}
	return out, total, nil
}

func (r *GormCourseOutcomeRepository) GetByID(ctx context.Context, id uint) (*domain.CourseOutcome, error) {
	var row courseOutcomeRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	o := rowToCourseOutcome(&row, nil)
	return &o, nil
}

func (r *GormCourseOutcomeRepository) Create(ctx context.Context, o *domain.CourseOutcome) error {
	row := courseOutcomeToRow(o)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	o.ID = row.ID
	o.CreatedAt = row.CreatedAt
	o.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormCourseOutcomeRepository) Save(ctx context.Context, o *domain.CourseOutcome) error {
	return r.db.WithContext(ctx).Save(courseOutcomeToRow(o)).Error
}

func (r *GormCourseOutcomeRepository) Delete(ctx context.Context, id uint) error {
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
	q := applyTaxonomyFilter(r.db.WithContext(ctx).Model(&courseSkillRow{}), filter)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []courseSkillRow
	if err := applyPagination(q, filter).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.CourseSkill, len(rows))
	for i := range rows {
		out[i] = rowToCourseSkill(&rows[i])
	}
	return out, total, nil
}

func (r *GormCourseSkillRepository) GetByID(ctx context.Context, id uint) (*domain.CourseSkill, error) {
	var row courseSkillRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	s := rowToCourseSkill(&row)
	return &s, nil
}

func (r *GormCourseSkillRepository) Create(ctx context.Context, s *domain.CourseSkill) error {
	row := courseSkillToRow(s)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	s.ID = row.ID
	s.CreatedAt = row.CreatedAt
	s.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormCourseSkillRepository) Save(ctx context.Context, s *domain.CourseSkill) error {
	return r.db.WithContext(ctx).Save(courseSkillToRow(s)).Error
}

func (r *GormCourseSkillRepository) Delete(ctx context.Context, id uint) error {
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

// NewGormCourseLevelRepository constructs a GormCourseLevelRepository.
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

func rowToCourseTopic(r *courseTopicRow, img *imageFileRow) domain.CourseTopic {
	child := []taxpkg.TreeNode(r.ChildTopics)
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	t := domain.CourseTopic{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		ImageFileID: r.ImageFileID, ChildTopics: child, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if img != nil {
		t.ImageFileURL = img.URL
		t.ImageFileKind = img.Kind
		t.ImageFileMime = img.Mime
	}
	return t
}

func courseTopicToRow(t *domain.CourseTopic) *courseTopicRow {
	child := t.ChildTopics
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return &courseTopicRow{
		ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
		ImageFileID: t.ImageFileID, ChildTopics: treeNodesJSONB(child),
		CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func rowToCourseOutcome(r *courseOutcomeRow, img *imageFileRow) domain.CourseOutcome {
	desc := []string(r.Description)
	if desc == nil {
		desc = []string{}
	}
	o := domain.CourseOutcome{
		ID: r.ID, ShortDescription: r.ShortDescription, Description: desc,
		Status: r.Status, ImageFileID: r.ImageFileID, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if img != nil {
		o.ImageFileURL = img.URL
		o.ImageFileKind = img.Kind
		o.ImageFileMime = img.Mime
	}
	return o
}

func courseOutcomeToRow(o *domain.CourseOutcome) *courseOutcomeRow {
	desc := o.Description
	if desc == nil {
		desc = []string{}
	}
	return &courseOutcomeRow{
		ID: o.ID, ShortDescription: o.ShortDescription, Description: descriptionJSONB(desc),
		Status: o.Status, ImageFileID: o.ImageFileID, CreatedBy: o.CreatedBy,
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
}

func rowToCourseSkill(r *courseSkillRow) domain.CourseSkill {
	child := []taxpkg.TreeNode(r.Children)
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return domain.CourseSkill{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Children: child, Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func courseSkillToRow(s *domain.CourseSkill) *courseSkillRow {
	child := s.Children
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return &courseSkillRow{
		ID: s.ID, Name: s.Name, Slug: s.Slug, Children: treeNodesJSONB(child),
		Status: s.Status, CreatedBy: s.CreatedBy, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
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
