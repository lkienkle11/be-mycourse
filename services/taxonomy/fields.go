package taxonomy

import (
	"strings"

	taxonomypkg "mycourse-io-be/pkg/taxonomy"
)

func trimmedTaxonomyFields(name, slug, status string) (string, string, string) {
	return strings.TrimSpace(name), strings.TrimSpace(slug), taxonomypkg.NormalizeTaxonomyStatus(status)
}

func applyOptionalTaxonomyNameSlugStatus(dstName, dstSlug *string, dstStatus *string, name, slug, status *string) {
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
