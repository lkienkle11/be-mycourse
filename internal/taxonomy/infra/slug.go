package infra

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	sharedslug "mycourse-io-be/internal/shared/slug"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/taxonomy/domain"
)

// maxTaxonomySlugLen mirrors every taxonomy root table's slug VARCHAR(255)
// column limit (migrations/000002_taxonomy_domain.up.sql,
// migrations/000009_taxonomy_topics_outcomes_skills.up.sql) and reuses the
// same 255 already relied on by tree node slug validation
// (internal/shared/taxonomy/tree_validate.go's DefaultMaxSlugLen), so root
// and tree-node slug length limits cannot drift apart.
const maxTaxonomySlugLen = taxpkg.DefaultMaxSlugLen

// taxonomySlugRetryAttempts mirrors internal/course/infra/repo_instructor.go's
// courseSlugCreateRetry (create) / updateSlugRetry (update) — same tuning
// rationale (random-suffix collisions can need more attempts under
// contention than a single pre-check allows).
const taxonomySlugRetryAttempts = 5

// taxonomySlugAvailable reports whether slug is free among active rows of
// model's table, optionally excluding one row id (used on update so
// resubmitting the current slug is never treated as a self-collision).
// Generalizes internal/course/infra/slug.go's courseSlugAvailable over the
// GORM row type, since 4 taxonomy tables share the same shape
// (slug + deleted_at) instead of course's single table.
func taxonomySlugAvailable[R any](ctx context.Context, db *gorm.DB, model *R, slug string, excludeID *string) (bool, error) {
	q := db.WithContext(ctx).Model(model).Where("deleted_at IS NULL").Where("slug = ?", slug)
	if excludeID != nil {
		if id := strings.TrimSpace(*excludeID); id != "" {
			q = q.Where("id != ?", id)
		}
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// isTaxonomySlugDuplicateKey reports whether err is a Postgres
// unique-violation on one of the 4 taxonomy slug partial unique indexes
// (migrations/000012_soft_delete_taxonomy_users_ban.up.sql). Mirrors
// internal/course/infra/repo_helpers.go's isCourseSlugDuplicateKey exactly
// (same message-substring detection — this repo's established way to
// recognize a duplicate-key error without a driver-specific type assertion).
func isTaxonomySlugDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "duplicate key") {
		return false
	}
	for _, idx := range []string{
		"uix_course_levels_slug_active",
		"uix_course_topics_slug_active",
		"uix_course_skills_slug_active",
		"uix_tags_slug_active",
	} {
		if strings.Contains(msg, idx) {
			return true
		}
	}
	return false
}

// buildTaxonomySuffixedCandidate truncates base to leave room for
// "-"+suffix within maxTaxonomySlugLen, erroring if no room remains at all —
// mirrors internal/course/infra/slug.go's buildSuffixedCandidate exactly
// (same guard against an unreachable leading-hyphen candidate, found by
// code review during course's own rework).
func buildTaxonomySuffixedCandidate(base, suffix string) (string, error) {
	truncatedBase := sharedslug.TruncateForSuffix(base, maxTaxonomySlugLen, len(suffix))
	if truncatedBase == "" {
		return "", fmt.Errorf("slug base %q leaves no room for a %d-character suffix within %d characters", base, len(suffix), maxTaxonomySlugLen)
	}
	return truncatedBase + "-" + suffix, nil
}

// resolveTaxonomyCreateSlug implements the create-time slug branch, shared
// by all 4 taxonomy root entities via the R type parameter for the target
// GORM row — same role as internal/course/infra/repo_instructor.go's
// resolveCreateSlug, generalized since taxonomy's 4 tables share one shape.
// manualSlug is "" for auto-generate; a non-empty value has already been
// trimmed+format-validated by the application layer. kind selects the
// auto-generation fallback prefix (see sharedslug.GenerateAutoSlugBase).
func resolveTaxonomyCreateSlug[R any](
	ctx context.Context, tx *gorm.DB, model *R,
	manualSlug, name string, kind sharedslug.Kind,
) (string, *domain.SlugConflictError, error) {
	if manualSlug != "" {
		available, err := taxonomySlugAvailable(ctx, tx, model, manualSlug, nil)
		if err != nil {
			return "", nil, err
		}
		if available {
			return manualSlug, nil, nil
		}
		recommended, err := sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
			candidate, err := buildTaxonomySuffixedCandidate(manualSlug, suffix)
			if err != nil {
				return "", false, err
			}
			ok, err := taxonomySlugAvailable(ctx, tx, model, candidate, nil)
			return candidate, ok, err
		})
		if err != nil {
			return "", nil, err
		}
		return "", &domain.SlugConflictError{RecommendedSlug: recommended}, nil
	}
	base, mustSuffix := sharedslug.GenerateAutoSlugBase(name, maxTaxonomySlugLen, kind)
	if !mustSuffix {
		available, err := taxonomySlugAvailable(ctx, tx, model, base, nil)
		if err != nil {
			return "", nil, err
		}
		if available {
			return base, nil, nil
		}
	}
	final, err := sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate, err := buildTaxonomySuffixedCandidate(base, suffix)
		if err != nil {
			return "", false, err
		}
		ok, err := taxonomySlugAvailable(ctx, tx, model, candidate, nil)
		return candidate, ok, err
	})
	return final, nil, err
}

// resolveTaxonomyUpdateSlug implements the update-time slug branch — no
// conflict-error path exists here (update auto-resolves directly via random
// suffix, per the product decision that taxonomy update mirrors course's
// UPDATE behavior, not its CREATE behavior). Also used for the "slug
// omitted" case: the caller passes the entity's own already-loaded,
// unchanged slug as newSlug, which this function's excludeID-scoped
// availability check always finds available (it belongs to this same row),
// returning it unchanged. Mirrors
// internal/course/infra/repo_instructor.go's resolveUpdateSlug.
func resolveTaxonomyUpdateSlug[R any](ctx context.Context, tx *gorm.DB, model *R, newSlug, excludeID string) (string, error) {
	available, err := taxonomySlugAvailable(ctx, tx, model, newSlug, &excludeID)
	if err != nil {
		return "", err
	}
	if available {
		return newSlug, nil
	}
	return sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate, err := buildTaxonomySuffixedCandidate(newSlug, suffix)
		if err != nil {
			return "", false, err
		}
		ok, err := taxonomySlugAvailable(ctx, tx, model, candidate, &excludeID)
		return candidate, ok, err
	})
}
