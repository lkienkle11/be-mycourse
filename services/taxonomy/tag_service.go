package taxonomy

import (
	"strings"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/helper"
	repo "mycourse-io-be/repository/taxonomy"
)

func ListTags(filter dto.TagFilter) ([]models.Tag, int64, error) {
	return repo.NewTagRepository(models.DB).ListTags(filter)
}

func CreateTag(actorID uint, req dto.CreateTagRequest) (*models.Tag, error) {
	row := &models.Tag{
		Tag: entities.Tag{
			Name:   strings.TrimSpace(req.Name),
			Slug:   strings.TrimSpace(req.Slug),
			Status: helper.NormalizeTaxonomyStatus(req.Status),
		},
	}
	if actorID > 0 {
		row.CreatedBy = &actorID
	}
	if err := repo.NewTagRepository(models.DB).CreateTag(row); err != nil {
		return nil, err
	}
	return row, nil
}

func UpdateTag(id uint, req dto.UpdateTagRequest) (*models.Tag, error) {
	r := repo.NewTagRepository(models.DB)
	row, err := r.GetTagByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		row.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		row.Status = helper.NormalizeTaxonomyStatus(*req.Status)
	}

	if err := r.UpdateTag(row); err != nil {
		return nil, err
	}
	return row, nil
}

func DeleteTag(id uint) error {
	return repo.NewTagRepository(models.DB).DeleteTag(id)
}
