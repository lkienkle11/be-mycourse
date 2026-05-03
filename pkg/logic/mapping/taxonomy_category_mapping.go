package mapping

import (
	"time"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
)

func ToCategoryResponse(row models.Category) dto.CategoryResponse {
	return dto.CategoryResponse{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		ImageURL:  row.ImageURL,
		Status:    string(row.Status),
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
	}
}

func ToCategoryResponses(rows []models.Category) []dto.CategoryResponse {
	out := make([]dto.CategoryResponse, 0, len(rows))
	for _, item := range rows {
		out = append(out, ToCategoryResponse(item))
	}
	return out
}
