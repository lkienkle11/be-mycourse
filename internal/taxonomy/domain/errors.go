package domain

import "errors"

var (
	// ErrTaxonomyOptimisticLock is returned when expected_row_version does not match.
	ErrTaxonomyOptimisticLock = errors.New("taxonomy resource was modified by another request; refresh and retry")
	// ErrTaxonomyCanonicalConflict is returned when canonical text disagrees with translations.en.
	ErrTaxonomyCanonicalConflict = errors.New("canonical field conflicts with translations.en")
)

// SlugConflictError is returned when a manually supplied slug on create
// already belongs to another active row of the same taxonomy entity type.
// Mirrors internal/course/domain.SlugConflictError: RecommendedSlug is
// confirmed available at construction time, but every write path re-checks
// at write time regardless (see internal/taxonomy/infra/slug.go).
type SlugConflictError struct {
	RecommendedSlug string
}

func (e *SlugConflictError) Error() string { return "taxonomy slug already exists" }
