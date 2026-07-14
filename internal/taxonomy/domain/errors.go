package domain

import "errors"

var (
	// ErrTaxonomyOptimisticLock is returned when expected_row_version does not match.
	ErrTaxonomyOptimisticLock = errors.New("taxonomy resource was modified by another request; refresh and retry")
	// ErrTaxonomyCanonicalConflict is returned when canonical text disagrees with translations.en.
	ErrTaxonomyCanonicalConflict = errors.New("canonical field conflicts with translations.en")
)
