package mapping

import (
	"strings"
	"time"

	taxonomypkg "mycourse-io-be/pkg/taxonomy"

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

// CategoryListHTTPPayload maps category rows to the list JSON payload (api/ must not import models — restrict_api).
func CategoryListHTTPPayload(rows []models.Category) any {
	return ToCategoryResponseModels(rows)
}

// CategoryRowHTTPPayload maps one category row to the create/update JSON payload.
func CategoryRowHTTPPayload(row models.Category) any {
	return ToCategoryResponseModel(row)
}

// TrimmedTaxonomyFields normalizes name/slug/status from string inputs (Rule 14).
func TrimmedTaxonomyFields(name, slug, status string) (string, string, string) {
	return strings.TrimSpace(name), strings.TrimSpace(slug), taxonomypkg.NormalizeTaxonomyStatus(status)
}

// ApplyOptionalTaxonomyNameSlugStatus applies optional dto pointers onto model string fields (Rule 14).
func ApplyOptionalTaxonomyNameSlugStatus(dstName, dstSlug, dstStatus *string, name, slug, status *string) {
	if name != nil {
		if v := strings.TrimSpace(*name); v != "" {
			*dstName = v
		}
	}
	if slug != nil {
		if v := strings.TrimSpace(*slug); v != "" {
			*dstSlug = v
		}
	}
	if status != nil && strings.TrimSpace(*status) != "" {
		*dstStatus = taxonomypkg.NormalizeTaxonomyStatus(*status)
	}
}

// CategoryModelForCreate builds a new GORM row from validated trimmed fields (Rule 14).
func CategoryModelForCreate(actorID uint, name, slug, statusNorm, imageFileID string) *models.Category {
	fid := imageFileID
	row := &models.Category{
		Name:        name,
		Slug:        slug,
		Status:      statusNorm,
		ImageFileID: &fid,
	}
	if actorID > 0 {
		row.CreatedBy = &actorID
	}
	return row
}
