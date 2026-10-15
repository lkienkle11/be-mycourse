package application

import (
	"errors"

	sharedslug "mycourse-io-be/internal/shared/slug"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/shared/validate"
)

// validateManualTaxonomySlug normalizes an optional manual slug on create via
// the shared parser (internal/shared/slug.ParseManual). An empty or
// whitespace-only input returns "" with no error - the repo resolves "" as
// "auto-generate".
func validateManualTaxonomySlug(raw string) (string, error) {
	validated, err := sharedslug.ParseManual(raw, taxpkg.DefaultMaxSlugLen)
	if err != nil {
		return "", errors.Join(ErrTaxonomyValidation, errors.New("slug format is invalid"))
	}
	return validated, nil
}

// resolveTaxonomySlugInput interprets an update request's Optional[string]
// slug field via the shared parser (internal/shared/slug.ParseUpdateInput).
// Returns (nil, nil) for "omitted - leave slug unchanged". Lives in the
// application layer because taxonomy's delivery layer is generic across all 5
// entity types.
func resolveTaxonomySlugInput(in validate.Optional[string]) (*string, error) {
	validated, err := sharedslug.ParseUpdateInput(in.Set, in.Value, taxpkg.DefaultMaxSlugLen)
	switch {
	case errors.Is(err, sharedslug.ErrEmpty):
		return nil, errors.Join(ErrTaxonomyValidation, errors.New("slug must not be null, empty, or whitespace-only"))
	case err != nil:
		return nil, errors.Join(ErrTaxonomyValidation, errors.New("slug format is invalid"))
	}
	return validated, nil
}
