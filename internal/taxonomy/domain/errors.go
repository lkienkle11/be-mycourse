package domain

import (
	"errors"

	"mycourse-io-be/internal/shared/slug"
)

var (
	// ErrTaxonomyOptimisticLock is returned when expected_row_version does not match.
	ErrTaxonomyOptimisticLock = errors.New("taxonomy resource was modified by another request; refresh and retry")
	// ErrTaxonomyCanonicalConflict is returned when canonical text disagrees with translations.en.
	ErrTaxonomyCanonicalConflict = errors.New("canonical field conflicts with translations.en")
)

// SlugConflictError is returned when a manually supplied slug on create
// already belongs to another active row of the same taxonomy entity type. It is
// the shared internal/shared/slug.ConflictError (one definition for course,
// taxonomy and user) exposed under the taxonomy domain name so callers keep
// using errors.As against domain.SlugConflictError.
type SlugConflictError = slug.ConflictError
