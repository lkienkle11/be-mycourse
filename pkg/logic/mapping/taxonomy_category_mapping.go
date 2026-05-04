package mapping

import (
	"time"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
)

func ToCategoryResponseModel(row models.Category) dto.CategoryResponse {
	return dto.CategoryResponse{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Image:     ToMediaFilePublicFromModel(row.ImageFile),
		Status:    string(row.Status),
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
	}
}

func ToCategoryResponseModels(rows []models.Category) []dto.CategoryResponse {
	out := make([]dto.CategoryResponse, 0, len(rows))
	for _, item := range rows {
		out = append(out, ToCategoryResponseModel(item))
	}
	return out
}
