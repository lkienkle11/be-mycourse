package mapping

import (
	"time"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/entities"
)

func ToTagResponse(row entities.Tag) dto.TagResponse {
	return dto.TagResponse{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Status:    string(row.Status),
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
	}
}

func ToTagResponses(rows []entities.Tag) []dto.TagResponse {
	out := make([]dto.TagResponse, 0, len(rows))
	for _, item := range rows {
		out = append(out, ToTagResponse(item))
	}
	return out
}
