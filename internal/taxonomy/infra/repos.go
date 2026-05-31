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
	sharedutils "mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/taxonomy/domain"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
)

// --- GORM row types ----------------------------------------------------------

type courseTopicRow struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	Name        string         `gorm:"size:255;not null"`
	Slug        string         `gorm:"size:255;not null;index"`
	ImageFileID *string        `gorm:"column:image_file_id;type:uuid"`
	ChildTopics treeNodesJSONB `gorm:"column:child_topics;type:jsonb;not null;default:'[]'"`
	Status      string         `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy   *uint          `gorm:"column:created_by"`
	CreatedAt   int64
	UpdatedAt   int64
	DeletedAt   *int64 `gorm:"column:deleted_at;index"`
}

func (courseTopicRow) TableName() string { return constants.TableTaxonomyCourseTopics }

type courseOutcomeRow struct {
	ID               uint             `gorm:"primaryKey;autoIncrement"`
	ShortDescription string           `gorm:"column:short_description;size:100;not null"`
	Description      descriptionJSONB `gorm:"type:jsonb;not null;default:'[]'"`
	ImageFileID      *string          `gorm:"column:image_file_id;type:uuid"`
	Status           string           `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy        *uint            `gorm:"column:created_by"`
	CreatedAt        int64
	UpdatedAt        int64
	DeletedAt        *int64 `gorm:"column:deleted_at;index"`
}

func (courseOutcomeRow) TableName() string { return constants.TableTaxonomyCourseOutcomes }

type courseSkillRow struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	Name      string         `gorm:"size:255;not null"`
	Slug      string         `gorm:"size:255;not null;index"`
	Children  treeNodesJSONB `gorm:"type:jsonb;not null;default:'[]'"`
	Status    string         `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy *uint          `gorm:"column:created_by"`
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64 `gorm:"column:deleted_at;index"`
}

func (courseSkillRow) TableName() string { return constants.TableTaxonomyCourseSkills }

type tagRow struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"size:255;not null"`
	Slug      string `gorm:"size:255;not null;index"`
	Status    string `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy *uint  `gorm:"column:created_by"`
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64 `gorm:"column:deleted_at;index"`
}

func (tagRow) TableName() string { return constants.TableTaxonomyTags }

type courseLevelRow struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"size:255;not null"`
	Slug      string `gorm:"size:255;not null;index"`
	Status    string `gorm:"type:taxonomy_status;not null;default:'ACTIVE'"`
	CreatedBy *uint  `gorm:"column:created_by"`
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64 `gorm:"column:deleted_at;index"`
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
	return listTaxonomyWithImageURLs(
		ctx,
		r.db,
		&courseTopicRow{},
		filter,
		applyTaxonomyFilter,
		mapCourseTopicRow,
		topicImageID,
		applyTopicImageRow,
	)
}

func (r *GormCourseTopicRepository) GetByID(ctx context.Context, id uint) (*domain.CourseTopic, error) {
	return taxonomyGetByIDWithImageURLs(
		ctx,
		r.db,
		id,
		mapCourseTopicRow,
		topicImageID,
		applyTopicImageRow,
	)
}

func (r *GormCourseTopicRepository) Create(ctx context.Context, t *domain.CourseTopic) error {
	return createTaxonomyDomain(ctx, r.db, t, courseTopicToRow, &t.ID, &t.CreatedAt, &t.UpdatedAt, metaCourseTopicRow)
}

func (r *GormCourseTopicRepository) Save(ctx context.Context, t *domain.CourseTopic) error {
	row := courseTopicToRow(t)
	gormx.TouchUpdated(&row.UpdatedAt)
	t.UpdatedAt = row.UpdatedAt
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormCourseTopicRepository) SoftDelete(ctx context.Context, id uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseTopicRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseTopicRepository) HardDelete(ctx context.Context, id uint) error {
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
	return listTaxonomyWithImageURLs(
		ctx,
		r.db,
		&courseOutcomeRow{},
		filter,
		applyOutcomeSearch,
		mapCourseOutcomeRow,
		outcomeImageID,
		applyOutcomeImageRow,
	)
}

func (r *GormCourseOutcomeRepository) GetByID(ctx context.Context, id uint) (*domain.CourseOutcome, error) {
	return taxonomyGetByIDWithImageURLs(
		ctx,
		r.db,
		id,
		mapCourseOutcomeRow,
		outcomeImageID,
		applyOutcomeImageRow,
	)
}

func (r *GormCourseOutcomeRepository) Create(ctx context.Context, o *domain.CourseOutcome) error {
	return createTaxonomyDomain(ctx, r.db, o, courseOutcomeToRow, &o.ID, &o.CreatedAt, &o.UpdatedAt, metaCourseOutcomeRow)
}

func (r *GormCourseOutcomeRepository) Save(ctx context.Context, o *domain.CourseOutcome) error {
	row := courseOutcomeToRow(o)
	gormx.TouchUpdated(&row.UpdatedAt)
	o.UpdatedAt = row.UpdatedAt
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormCourseOutcomeRepository) SoftDelete(ctx context.Context, id uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseOutcomeRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseOutcomeRepository) HardDelete(ctx context.Context, id uint) error {
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
	return taxonomyList(ctx, r.db, &courseSkillRow{}, filter, applyTaxonomyFilter, rowToCourseSkill)
}

func (r *GormCourseSkillRepository) GetByID(ctx context.Context, id uint) (*domain.CourseSkill, error) {
	return taxonomyGetByID(ctx, r.db, id, rowToCourseSkill)
}

func (r *GormCourseSkillRepository) Create(ctx context.Context, s *domain.CourseSkill) error {
	return createTaxonomyDomain(ctx, r.db, s, courseSkillToRow, &s.ID, &s.CreatedAt, &s.UpdatedAt, metaCourseSkillRow)
}

func (r *GormCourseSkillRepository) Save(ctx context.Context, s *domain.CourseSkill) error {
	row := courseSkillToRow(s)
	gormx.TouchUpdated(&row.UpdatedAt)
	s.UpdatedAt = row.UpdatedAt
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormCourseSkillRepository) SoftDelete(ctx context.Context, id uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseSkillRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseSkillRepository) HardDelete(ctx context.Context, id uint) error {
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
	return taxonomyList(ctx, r.db, &tagRow{}, filter, applyTaxonomyFilter, rowToTag)
}

func (r *GormTagRepository) GetByID(ctx context.Context, id uint) (*domain.Tag, error) {
	return taxonomyGetByID(ctx, r.db, id, rowToTag)
}

func (r *GormTagRepository) Create(ctx context.Context, t *domain.Tag) error {
	return createTaxonomyDomain(ctx, r.db, t, tagToRow, &t.ID, &t.CreatedAt, &t.UpdatedAt, metaTagRow)
}

func (r *GormTagRepository) Save(ctx context.Context, t *domain.Tag) error {
	row := tagToRow(t)
	gormx.TouchUpdated(&row.UpdatedAt)
	t.UpdatedAt = row.UpdatedAt
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormTagRepository) SoftDelete(ctx context.Context, id uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &tagRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormTagRepository) HardDelete(ctx context.Context, id uint) error {
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
	return taxonomyList(ctx, r.db, &courseLevelRow{}, filter, applyTaxonomyFilter, rowToCourseLevel)
}

func (r *GormCourseLevelRepository) GetByID(ctx context.Context, id uint) (*domain.CourseLevel, error) {
	return taxonomyGetByID(ctx, r.db, id, rowToCourseLevel)
}

func (r *GormCourseLevelRepository) Create(ctx context.Context, cl *domain.CourseLevel) error {
	return createTaxonomyDomain(ctx, r.db, cl, courseLevelToRow, &cl.ID, &cl.CreatedAt, &cl.UpdatedAt, metaCourseLevelRow)
}

func (r *GormCourseLevelRepository) Save(ctx context.Context, cl *domain.CourseLevel) error {
	row := courseLevelToRow(cl)
	gormx.TouchUpdated(&row.UpdatedAt)
	cl.UpdatedAt = row.UpdatedAt
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormCourseLevelRepository) SoftDelete(ctx context.Context, id uint) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &courseLevelRow{}, "id = ? AND deleted_at IS NULL", id)
}

func (r *GormCourseLevelRepository) HardDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&courseLevelRow{}, id).Error
}

// --- row mappers -------------------------------------------------------------

type imageFileRow struct {
	ID   string `gorm:"column:id"`
	URL  string `gorm:"column:url"`
	Kind string `gorm:"column:kind"`
	Mime string `gorm:"column:mime_type"`
}

func topicImageID(row *domain.CourseTopic) *string { return row.ImageFileID }

func outcomeImageID(row *domain.CourseOutcome) *string { return row.ImageFileID }

func applyTopicImageRow(row *domain.CourseTopic, img imageFileRow) {
	row.ImageFileURL = img.URL
	row.ImageFileKind = img.Kind
	row.ImageFileMime = img.Mime
}

func applyOutcomeImageRow(row *domain.CourseOutcome, img imageFileRow) {
	row.ImageFileURL = img.URL
	row.ImageFileKind = img.Kind
	row.ImageFileMime = img.Mime
}

func trimStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func collectImageFileIDs[T any](rows []T, getID func(*T) *string) []string {
	ids := make([]string, 0, len(rows))
	seen := map[string]struct{}{}
	for i := range rows {
		id := trimStringPtr(getID(&rows[i]))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func loadImageFileMap(ctx context.Context, db *gorm.DB, ids []string) (map[string]imageFileRow, error) {
	if len(ids) == 0 {
		return map[string]imageFileRow{}, nil
	}
	var rows []imageFileRow
	err := db.WithContext(ctx).
		Table(constants.TableMediaFiles).
		Select("id, url, kind, mime_type").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]imageFileRow, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func hydrateImageURLs[T any](
	ctx context.Context,
	db *gorm.DB,
	rows []T,
	getID func(*T) *string,
	setImage func(*T, imageFileRow),
) ([]T, error) {
	ids := collectImageFileIDs(rows, getID)
	images, err := loadImageFileMap(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		id := trimStringPtr(getID(&rows[i]))
		img, ok := images[id]
		if !ok {
			continue
		}
		setImage(&rows[i], img)
	}
	return rows, nil
}

func listTaxonomyWithImageURLs[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	model *R,
	filter domain.TaxonomyFilter,
	applySearch func(*gorm.DB, domain.TaxonomyFilter) *gorm.DB,
	mapRow func(*R) D,
	getID func(*D) *string,
	setImage func(*D, imageFileRow),
) ([]D, int64, error) {
	rows, total, err := taxonomyList(ctx, db, model, filter, applySearch, mapRow)
	if err != nil {
		return nil, 0, err
	}
	rows, err = hydrateImageURLs(ctx, db, rows, getID, setImage)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func taxonomyGetByIDWithImageURLs[R any, D any](
	ctx context.Context,
	db *gorm.DB,
	id uint,
	mapRow func(*R) D,
	getID func(*D) *string,
	setImage func(*D, imageFileRow),
) (*D, error) {
	row, err := taxonomyGetByID(ctx, db, id, mapRow)
	if err != nil {
		return nil, err
	}
	rows, err := hydrateImageURLs(ctx, db, []D{*row}, getID, setImage)
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func rowToCourseTopic(r *courseTopicRow, img *imageFileRow) domain.CourseTopic {
	child := []taxpkg.TreeNode(r.ChildTopics)
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	t := domain.CourseTopic{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		ImageFileID: r.ImageFileID, ChildTopics: child, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
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
		CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, DeletedAt: t.DeletedAt,
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
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
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
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt, DeletedAt: o.DeletedAt,
	}
}

func rowToCourseSkill(r *courseSkillRow) domain.CourseSkill {
	child := []taxpkg.TreeNode(r.Children)
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return domain.CourseSkill{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Children: child, Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
	}
}

func courseSkillToRow(s *domain.CourseSkill) *courseSkillRow {
	child := s.Children
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return &courseSkillRow{
		ID: s.ID, Name: s.Name, Slug: s.Slug, Children: treeNodesJSONB(child),
		Status: s.Status, CreatedBy: s.CreatedBy, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt, DeletedAt: s.DeletedAt,
	}
}

func rowToTag(r *tagRow) domain.Tag {
	return domain.Tag{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
	}
}

func tagToRow(t *domain.Tag) *tagRow {
	return &tagRow{
		ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
		CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, DeletedAt: t.DeletedAt,
	}
}

func rowToCourseLevel(r *courseLevelRow) domain.CourseLevel {
	return domain.CourseLevel{
		ID: r.ID, Name: r.Name, Slug: r.Slug, Status: r.Status,
		CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, DeletedAt: r.DeletedAt,
	}
}

func courseLevelToRow(cl *domain.CourseLevel) *courseLevelRow {
	return &courseLevelRow{
		ID: cl.ID, Name: cl.Name, Slug: cl.Slug, Status: cl.Status,
		CreatedBy: cl.CreatedBy, CreatedAt: cl.CreatedAt, UpdatedAt: cl.UpdatedAt, DeletedAt: cl.DeletedAt,
	}
}
