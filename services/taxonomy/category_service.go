package taxonomy

import (
	"strings"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	repo "mycourse-io-be/repository/taxonomy"
	mediasvc "mycourse-io-be/services/media"
)

func ListCategories(filter dto.CategoryFilter) ([]entities.Category, int64, error) {
	rows, total, err := repo.NewCategoryRepository(models.DB).ListCategories(filter)
	if err != nil {
		return nil, 0, err
	}
	return categoryEntities(rows), total, nil
}

func CreateCategory(actorID uint, req dto.CreateCategoryRequest) (*entities.Category, error) {
	n, s, st := trimmedTaxonomyFields(req.Name, req.Slug, req.Status)
	row := &models.Category{
		Category: entities.Category{
			Name:     n,
			Slug:     s,
			ImageURL: strings.TrimSpace(req.ImageURL),
			Status:   st,
		},
	}
	if actorID > 0 {
		row.CreatedBy = &actorID
	}
	if err := repo.NewCategoryRepository(models.DB).CreateCategory(row); err != nil {
		return nil, err
	}
	return &row.Category, nil
}

func UpdateCategory(id uint, req dto.UpdateCategoryRequest) (*entities.Category, error) {
	r := repo.NewCategoryRepository(models.DB)
	row, err := r.GetCategoryByID(id)
	if err != nil {
		return nil, err
	}

	applyOptionalTaxonomyNameSlugStatus(&row.Name, &row.Slug, &row.Status, req.Name, req.Slug, req.Status)
	prevImageURL := row.ImageURL

	if req.ImageURL != nil {
		row.ImageURL = strings.TrimSpace(*req.ImageURL)
	}

	if err := r.UpdateCategory(row); err != nil {
		return nil, err
	}

	// Enqueue orphan cleanup for superseded image_url after successful DB commit.
	if req.ImageURL != nil && prevImageURL != "" && prevImageURL != row.ImageURL {
		mediasvc.EnqueueOrphanImageCleanup(prevImageURL)
	}

	return &row.Category, nil
}

func DeleteCategory(id uint) error {
	r := repo.NewCategoryRepository(models.DB)
	row, err := r.GetCategoryByID(id)
	if err != nil {
		return err
	}
	if err := r.DeleteCategory(id); err != nil {
		return err
	}
	// Enqueue orphan cleanup for image_url after successful DB delete.
	if row.ImageURL != "" {
		mediasvc.EnqueueOrphanImageCleanup(row.ImageURL)
	}
	return nil
}
