package taxonomy

import (
	"strings"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/helper"
	repo "mycourse-io-be/repository/taxonomy"
)

func ListCategories(filter dto.CategoryFilter) ([]models.Category, int64, error) {
	return repo.NewCategoryRepository(models.DB).ListCategories(filter)
}

func CreateCategory(actorID uint, req dto.CreateCategoryRequest) (*models.Category, error) {
	row := &models.Category{
		Category: entities.Category{
			Name:     strings.TrimSpace(req.Name),
			Slug:     strings.TrimSpace(req.Slug),
			ImageURL: strings.TrimSpace(req.ImageURL),
			Status:   helper.NormalizeTaxonomyStatus(req.Status),
		},
	}
	if actorID > 0 {
		row.CreatedBy = &actorID
	}
	if err := repo.NewCategoryRepository(models.DB).CreateCategory(row); err != nil {
		return nil, err
	}
	return row, nil
}

func UpdateCategory(id uint, req dto.UpdateCategoryRequest) (*models.Category, error) {
	r := repo.NewCategoryRepository(models.DB)
	row, err := r.GetCategoryByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		row.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.ImageURL != nil {
		row.ImageURL = strings.TrimSpace(*req.ImageURL)
	}
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		row.Status = helper.NormalizeTaxonomyStatus(*req.Status)
	}

	if err := r.UpdateCategory(row); err != nil {
		return nil, err
	}
	return row, nil
}

func DeleteCategory(id uint) error {
	return repo.NewCategoryRepository(models.DB).DeleteCategory(id)
}
