package taxonomy

import (
	"mycourse-io-be/pkg/logic/mapping"
)

func trimmedTaxonomyFields(name, slug, status string) (string, string, string) {
	return mapping.TrimmedTaxonomyFields(name, slug, status)
}

func applyOptionalTaxonomyNameSlugStatus(dstName, dstSlug *string, dstStatus *string, name, slug, status *string) {
	mapping.ApplyOptionalTaxonomyNameSlugStatus(dstName, dstSlug, dstStatus, name, slug, status)
}
