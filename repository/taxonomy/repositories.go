package taxonomy

import (
	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) ListCategories(filter dto.CategoryFilter) ([]models.Category, int64, error) {
	q, total, err := taxonomyFilteredQuery(r.db.Preload("ImageFile"), &models.Category{}, filter.Status, filter.BaseFilter, taxonomyListSearchCols)
	if err != nil {
		return nil, 0, err
	}
	rows, err := taxonomyOrderedFind[models.Category](q, filter.BaseFilter, taxonomyListSortCols)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *CategoryRepository) CreateCategory(row *models.Category) error {
	return gormCreateRow(r.db, row)
}

func (r *CategoryRepository) GetCategoryByID(id uint) (*models.Category, error) {
	var row models.Category
	if err := r.db.Preload("ImageFile").First(&row, id).Error; err != nil {
		return nil, pkgerrors.MapRecordNotFound(err)
	}
	return &row, nil
}

func (r *CategoryRepository) UpdateCategory(row *models.Category) error {
	return gormSaveRow(r.db, row)
}

func (r *CategoryRepository) DeleteCategory(id uint) error {
	return gormDeleteModelByID[models.Category](r.db, id)
}

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) ListTags(filter dto.TagFilter) ([]models.Tag, int64, error) {
	return listTaxonomyModels[models.Tag](r.db, &models.Tag{}, filter.Status, filter.BaseFilter)
}

func (r *TagRepository) CreateTag(row *models.Tag) error {
	return gormCreateRow(r.db, row)
}

func (r *TagRepository) GetTagByID(id uint) (*models.Tag, error) {
	return gormGetByID[models.Tag](r.db, id)
}

func (r *TagRepository) UpdateTag(row *models.Tag) error {
	return gormSaveRow(r.db, row)
}

func (r *TagRepository) DeleteTag(id uint) error {
	return gormDeleteModelByID[models.Tag](r.db, id)
}

type CourseLevelRepository struct {
	db *gorm.DB
}

func NewCourseLevelRepository(db *gorm.DB) *CourseLevelRepository {
	return &CourseLevelRepository{db: db}
}

func (r *CourseLevelRepository) ListCourseLevels(filter dto.CourseLevelFilter) ([]models.CourseLevel, int64, error) {
	return listTaxonomyModels[models.CourseLevel](r.db, &models.CourseLevel{}, filter.Status, filter.BaseFilter)
}

func (r *CourseLevelRepository) CreateCourseLevel(row *models.CourseLevel) error {
	return gormCreateRow(r.db, row)
}

func (r *CourseLevelRepository) GetCourseLevelByID(id uint) (*models.CourseLevel, error) {
	return gormGetByID[models.CourseLevel](r.db, id)
}

func (r *CourseLevelRepository) UpdateCourseLevel(row *models.CourseLevel) error {
	return gormSaveRow(r.db, row)
}

func (r *CourseLevelRepository) DeleteCourseLevel(id uint) error {
	return gormDeleteModelByID[models.CourseLevel](r.db, id)
}
