package infra

import (
	stderrors "errors"
	"testing"
)

func TestIsTaxonomySlugDuplicateKey(t *testing.T) {
	for _, idx := range []string{
		"uix_course_levels_slug_active",
		"uix_course_topics_slug_active",
		"uix_course_skills_slug_active",
		"uix_tags_slug_active",
	} {
		err := stderrors.New(`ERROR: duplicate key value violates unique constraint "` + idx + `"`)
		if !isTaxonomySlugDuplicateKey(err) {
			t.Fatalf("expected duplicate slug detection for index %q", idx)
		}
	}
	if isTaxonomySlugDuplicateKey(nil) {
		t.Fatal("nil should not match")
	}
	unrelated := stderrors.New(`ERROR: duplicate key value violates unique constraint "uix_courses_slug_active"`)
	if isTaxonomySlugDuplicateKey(unrelated) {
		t.Fatal("an unrelated table's duplicate-key message should not match")
	}
}

// Auto-generation (internal/shared/slug.GenerateAutoSlugBase), suffix candidate
// building (slug.BuildSuffixedCandidate) and the create/update resolvers
// (slug.ResolveCreate / slug.ResolveUpdate) are shared by course, taxonomy and
// user, and are covered by the tests in internal/shared/slug.
