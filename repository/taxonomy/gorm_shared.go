package taxonomy

import (
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/dto"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/query"
)

// Shared column maps for tag / category / course_level list endpoints.
var taxonomyListSearchCols = map[string]string{"name": "name", "slug": "slug"}

var taxonomyListSortCols = map[string]string{
	"id": "id", "name": "name", "slug": "slug", "status": "status", "created_at": "created_at",
}

func taxonomyFilteredQuery(db *gorm.DB, model any, status *string, base dto.BaseFilter, searchCols map[string]string) (*gorm.DB, int64, error) {
	q := db.Model(model)
	if status != nil && strings.TrimSpace(*status) != "" {
		q = q.Where("status = ?", strings.ToUpper(strings.TrimSpace(*status)))
	}
	if where, arg, ok := query.BuildSearchClause(base, searchCols); ok {
		q = q.Where(where, arg)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	return q, total, nil
}

func taxonomyOrderedFind[T any](q *gorm.DB, base dto.BaseFilter, sortCols map[string]string) ([]T, error) {
	p := query.ParseListFilter(base)
	sortClause := query.BuildSortClause(base, sortCols, "id")
	var rows []T
	if err := q.Order(sortClause).Offset(p.Offset).Limit(p.PerPage).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func gormCreateRow(db *gorm.DB, row any) error {
	return db.Create(row).Error
}

func gormSaveRow(db *gorm.DB, row any) error {
	return db.Save(row).Error
}

func gormGetByID[T any](db *gorm.DB, id uint) (*T, error) {
	var row T
	if err := db.First(&row, id).Error; err != nil {
		return nil, pkgerrors.MapRecordNotFound(err)
	}
	return &row, nil
}

func gormDeleteModelByID[T any](db *gorm.DB, id uint) error {
	var placeholder T
	return db.Delete(&placeholder, id).Error
}

func listTaxonomyModels[T any](db *gorm.DB, model any, status *string, base dto.BaseFilter) ([]T, int64, error) {
	q, total, err := taxonomyFilteredQuery(db, model, status, base, taxonomyListSearchCols)
	if err != nil {
		return nil, 0, err
	}
	rows, err := taxonomyOrderedFind[T](q, base, taxonomyListSortCols)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
