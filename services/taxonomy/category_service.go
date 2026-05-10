package taxonomy

import (
	"strings"

	"mycourse-io-be/dto"
	jobmedia "mycourse-io-be/internal/jobs/media"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/logic/mapping"
	repo "mycourse-io-be/repository/taxonomy"
	mediasvc "mycourse-io-be/services/media"
)

func ListCategories(filter dto.CategoryFilter) ([]models.Category, int64, error) {
	rows, total, err := repo.NewCategoryRepository(models.DB).ListCategories(filter)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func CreateCategory(actorID uint, req dto.CreateCategoryRequest) (*models.Category, error) {
	fileID := strings.TrimSpace(req.ImageFileID)
	if _, err := mediasvc.LoadValidatedProfileImageFile(fileID); err != nil {
		return nil, err
	}
	n, s, st := mapping.TrimmedTaxonomyFields(req.Name, req.Slug, req.Status)
	fid := fileID
	row := mapping.CategoryModelForCreate(actorID, n, s, st, fid)
	r := repo.NewCategoryRepository(models.DB)
	if err := r.CreateCategory(row); err != nil {
		return nil, err
	}
	out, err := r.GetCategoryByID(row.ID)
	if err != nil {
		return row, nil
	}
	return out, nil
}

func mutateCategoryImageFileID(row *models.Category, imageFileID *string) error {
	if imageFileID == nil {
		return nil
	}
	next := strings.TrimSpace(*imageFileID)
	if next == "" {
		row.ImageFileID = nil
		return nil
	}
	if _, err := mediasvc.LoadValidatedProfileImageFile(next); err != nil {
		return err
	}
	row.ImageFileID = &next
	return nil
}

func categoryImageFileIDString(row *models.Category) string {
	if row.ImageFileID == nil {
		return ""
	}
	return strings.TrimSpace(*row.ImageFileID)
}

func UpdateCategory(id uint, req dto.UpdateCategoryRequest) (*models.Category, error) {
	r := repo.NewCategoryRepository(models.DB)
	row, err := r.GetCategoryByID(id)
	if err != nil {
		return nil, err
	}

	mapping.ApplyOptionalTaxonomyNameSlugStatus(&row.Name, &row.Slug, &row.Status, req.Name, req.Slug, req.Status)

	prevFileID := categoryImageFileIDString(row)
	if err := mutateCategoryImageFileID(row, req.ImageFileID); err != nil {
		return nil, err
	}

	if err := r.UpdateCategory(row); err != nil {
		return nil, err
	}

	nextFileID := categoryImageFileIDString(row)
	if req.ImageFileID != nil && prevFileID != "" && prevFileID != nextFileID {
		jobmedia.EnqueueOrphanCleanupForMediaFileID(prevFileID)
	}

	out, err := r.GetCategoryByID(id)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func DeleteCategory(id uint) error {
	r := repo.NewCategoryRepository(models.DB)
	row, err := r.GetCategoryByID(id)
	if err != nil {
		return err
	}
	imageFID := categoryImageFileIDString(row)
	if err := r.DeleteCategory(id); err != nil {
		return err
	}
	if imageFID != "" {
		jobmedia.EnqueueOrphanCleanupForMediaFileID(imageFID)
	}
	return nil
}
