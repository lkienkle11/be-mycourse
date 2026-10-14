package application

import (
	"errors"
	"strings"

	sharedslug "mycourse-io-be/internal/shared/slug"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/shared/validate"
)

// validateManualTaxonomySlug trims raw and format-validates it, mirroring
// internal/course/application/service.go's manual-slug handling inside
// CreateCourse. An empty/whitespace-only input (after trimming) returns ""
// with no error — the caller's repo.Create resolves "" as "auto-generate."
func validateManualTaxonomySlug(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	validated, ok := sharedslug.ValidateManualFormat(trimmed, taxpkg.DefaultMaxSlugLen)
	if !ok {
		return "", errors.Join(ErrTaxonomyValidation, errors.New("slug format is invalid"))
	}
	return validated, nil
}

// resolveTaxonomySlugInput interprets an update request's Optional[string]
// slug field, mirroring internal/course/delivery/handler_instructor.go's
// updateBasicInfo slug block, relocated to the application layer here
// because taxonomy's delivery layer is generic/shared across all 5 entity
// types (updateTaxonomyMutation) while course's is not. Returns (nil, nil)
// for "omitted — leave slug unchanged."
func resolveTaxonomySlugInput(in validate.Optional[string]) (*string, error) {
	if !in.Set {
		return nil, nil
	}
	trimmed := strings.TrimSpace(in.Value)
	if trimmed == "" {
		return nil, errors.Join(ErrTaxonomyValidation, errors.New("slug must not be null, empty, or whitespace-only"))
	}
	validated, ok := sharedslug.ValidateManualFormat(trimmed, taxpkg.DefaultMaxSlugLen)
	if !ok {
		return nil, errors.Join(ErrTaxonomyValidation, errors.New("slug format is invalid"))
	}
	return &validated, nil
}
