package infra

import (
	stderrors "errors"
	"testing"
)

func TestIsCourseSlugDuplicateKey(t *testing.T) {
	err := stderrors.New(`ERROR: duplicate key value violates unique constraint "uix_courses_slug_active"`)
	if !isCourseSlugDuplicateKey(err) {
		t.Fatal("expected duplicate slug detection")
	}
	if isCourseSlugDuplicateKey(nil) {
		t.Fatal("nil should not match")
	}
}

// Auto-generation (internal/shared/slug.GenerateAutoSlugBase), suffix candidate
// building (slug.BuildSuffixedCandidate) and the create/update resolvers
// (slug.ResolveCreate / slug.ResolveUpdate) are shared by course, taxonomy and
// user, and are covered by the tests in internal/shared/slug.
