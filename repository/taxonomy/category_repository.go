package taxonomy

import (
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/query"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) ListCategories(filter dto.CategoryFilter) ([]models.Category, int64, error) {
	q := r.db.Model(&models.Category{})

	if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
		q = q.Where("status = ?", strings.ToUpper(strings.TrimSpace(*filter.Status)))
	}

	if where, arg, ok := query.BuildSearchClause(filter.BaseFilter, map[string]string{
		"name": "name",
		"slug": "slug",
	}); ok {
		q = q.Where(where, arg)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	p := query.ParseListFilter(filter.BaseFilter)
	sortClause := query.BuildSortClause(filter.BaseFilter, map[string]string{
		"id":         "id",
		"name":       "name",
		"slug":       "slug",
		"status":     "status",
		"created_at": "created_at",
	}, "id")

	var rows []models.Category
	if err := q.Order(sortClause).Offset(p.Offset).Limit(p.PerPage).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *CategoryRepository) CreateCategory(row *models.Category) error {
	return r.db.Create(row).Error
}

func (r *CategoryRepository) GetCategoryByID(id uint) (*models.Category, error) {
	var row models.Category
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CategoryRepository) UpdateCategory(row *models.Category) error {
	return r.db.Save(row).Error
}

func (r *CategoryRepository) DeleteCategory(id uint) error {
	return r.db.Delete(&models.Category{}, id).Error
}
