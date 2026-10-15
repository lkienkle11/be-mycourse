package infra

import (
	stderrors "errors"
	"strings"
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

// generateAutoTaxonomySlugBase was unified into
// internal/shared/slug.GenerateAutoSlugBase (see
// internal/shared/slug/autobase_test.go for its "normal name" /
// "non-Latin transliteration" / "emoji-only fallback per kind" /
// "truncation" coverage, now shared across course, taxonomy root entities,
// and tree nodes instead of duplicated per bounded context).

func TestBuildTaxonomySuffixedCandidateNeverExceedsMaxLen(t *testing.T) {
	base := strings.Repeat("a", maxTaxonomySlugLen)
	for suffixLen := 7; suffixLen <= 10; suffixLen++ {
		suffix := strings.Repeat("x", suffixLen)
		candidate, err := buildTaxonomySuffixedCandidate(base, suffix)
		if err != nil {
			t.Fatalf("suffixLen=%d: unexpected error: %v", suffixLen, err)
		}
		if len(candidate) > maxTaxonomySlugLen {
			t.Fatalf("suffixLen=%d: candidate length %d exceeds maxTaxonomySlugLen %d (candidate=%q)", suffixLen, len(candidate), maxTaxonomySlugLen, candidate)
		}
		if strings.HasPrefix(candidate, "-") {
			t.Fatalf("suffixLen=%d: candidate %q starts with a hyphen — invalid slug format", suffixLen, candidate)
		}
	}
}

// TestBuildTaxonomySuffixedCandidateErrorsWhenNoRoomForBase mirrors course's
// own equivalent regression guard (internal/course/infra/repo_slug_test.go's
// TestBuildSuffixedCandidateErrorsWhenNoRoomForBase): once suffix escalation
// leaves no room for even a 1-character base, this must error instead of
// silently producing a leading-hyphen candidate that violates slug format.
func TestBuildTaxonomySuffixedCandidateErrorsWhenNoRoomForBase(t *testing.T) {
	suffix := strings.Repeat("x", maxTaxonomySlugLen)
	_, err := buildTaxonomySuffixedCandidate("any-base", suffix)
	if err == nil {
		t.Fatal("expected an error when the suffix alone leaves no room for any base")
	}
}
