package mapping

import (
	"time"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
)

func ToTagResponseModel(row models.Tag) dto.TagResponse {
	return dto.TagResponse{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Status:    row.Status,
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
	}
}

func ToTagResponsesFromModels(rows []models.Tag) []dto.TagResponse {
	out := make([]dto.TagResponse, len(rows))
	for i := range rows {
		out[i] = ToTagResponseModel(rows[i])
	}
	return out
}

func ToCourseLevelResponseModel(row models.CourseLevel) dto.CourseLevelResponse {
	return dto.CourseLevelResponse{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Status:    row.Status,
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
	}
}

func ToCourseLevelResponsesFromModels(rows []models.CourseLevel) []dto.CourseLevelResponse {
	out := make([]dto.CourseLevelResponse, len(rows))
	for i := range rows {
		out[i] = ToCourseLevelResponseModel(rows[i])
	}
	return out
}

// TagListHTTPPayload maps tag rows to the list JSON payload (api/ must not import models).
func TagListHTTPPayload(rows []models.Tag) any {
	return ToTagResponsesFromModels(rows)
}

// TagRowHTTPPayload maps one tag row to the create/update JSON payload.
func TagRowHTTPPayload(row models.Tag) any {
	return ToTagResponseModel(row)
}

// CourseLevelListHTTPPayload maps course level rows to the list JSON payload.
func CourseLevelListHTTPPayload(rows []models.CourseLevel) any {
	return ToCourseLevelResponsesFromModels(rows)
}

// CourseLevelRowHTTPPayload maps one course level row to the create/update JSON payload.
func CourseLevelRowHTTPPayload(row models.CourseLevel) any {
	return ToCourseLevelResponseModel(row)
}
