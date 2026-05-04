package taxonomy

import (
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	taxonomypkg "mycourse-io-be/pkg/taxonomy"
)

func trimmedTaxonomyFields(name, slug, status string) (string, string, constants.TaxonomyStatus) {
	return strings.TrimSpace(name), strings.TrimSpace(slug), taxonomypkg.NormalizeTaxonomyStatus(status)
}

func applyOptionalTaxonomyNameSlugStatus(dstName, dstSlug *string, dstStatus *constants.TaxonomyStatus, name, slug, status *string) {
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

func categoryEntities(rows []models.Category) []entities.Category {
	out := make([]entities.Category, len(rows))
	for i := range rows {
		out[i] = rows[i].Category
	}
	return out
}

func tagEntities(rows []models.Tag) []entities.Tag {
	out := make([]entities.Tag, len(rows))
	for i := range rows {
		out[i] = rows[i].Tag
	}
	return out
}

func courseLevelEntities(rows []models.CourseLevel) []entities.CourseLevel {
	out := make([]entities.CourseLevel, len(rows))
	for i := range rows {
		out[i] = rows[i].CourseLevel
	}
	return out
}
