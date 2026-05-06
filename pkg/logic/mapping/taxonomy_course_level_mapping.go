package mapping

import (
	"time"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/entities"
)

func ToCourseLevelResponse(row entities.CourseLevel) dto.CourseLevelResponse {
	return dto.CourseLevelResponse{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Status:    string(row.Status),
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
	}
}

func ToCourseLevelResponses(rows []entities.CourseLevel) []dto.CourseLevelResponse {
	out := make([]dto.CourseLevelResponse, 0, len(rows))
	for _, item := range rows {
		out = append(out, ToCourseLevelResponse(item))
	}
	return out
}
