package infra

import (
	"context"

	"gorm.io/gorm"

	sharedslug "mycourse-io-be/internal/shared/slug"
	"mycourse-io-be/internal/shared/slug/gormslug"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
)

// maxTaxonomySlugLen mirrors every taxonomy root table's slug VARCHAR(255)
// column limit and reuses the same 255 relied on by tree node slug validation
// (internal/shared/taxonomy/tree_validate.go's DefaultMaxSlugLen), so root and
// tree-node slug length limits cannot drift apart.
const maxTaxonomySlugLen = taxpkg.DefaultMaxSlugLen

// taxonomySlugIndexNames are the 4 taxonomy root tables' partial unique slug
// indexes (migrations/000012_soft_delete_taxonomy_users_ban.up.sql).
var taxonomySlugIndexNames = []string{
	"uix_course_levels_slug_active",
	"uix_course_topics_slug_active",
	"uix_course_skills_slug_active",
	"uix_tags_slug_active",
}

// isTaxonomySlugDuplicateKey reports whether err is a unique violation on one
// of the taxonomy slug indexes, so only a genuine slug race is retried.
func isTaxonomySlugDuplicateKey(err error) bool {
	return gormslug.IsDuplicate(err, taxonomySlugIndexNames...)
}

// resolveTaxonomyUpdateSlug resolves the slug to store on update through the
// shared resolver (internal/shared/slug.ResolveUpdate) against model's table,
// excluding the row's own id. Update never reports a conflict: a taken slug is
// auto-suffixed. Also used for the "slug omitted" case, where the caller
// passes the row's own unchanged slug, which the excludeID-scoped check always
// finds available.
func resolveTaxonomyUpdateSlug[R any](ctx context.Context, tx *gorm.DB, model *R, newSlug, excludeID string) (string, error) {
	return sharedslug.ResolveUpdate(newSlug, maxTaxonomySlugLen,
		gormslug.AvailableIn(ctx, tx, func() *R { return model }, &excludeID))
}
