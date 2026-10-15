package infra

import (
	stderrors "errors"
	"strings"
	"testing"

	"mycourse-io-be/internal/course/domain"
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

// generateAutoSlugBase was unified into internal/shared/slug.GenerateAutoSlugBase
// (see internal/shared/slug/autobase_test.go for its "normal title" /
// "non-Latin transliteration" / "emoji-only fallback" / "truncation" coverage,
// now shared across course, taxonomy root entities, and tree nodes instead of
// duplicated per bounded context).

func TestBuildSuffixedCandidateNeverExceedsMaxSlugLen(t *testing.T) {
	base := strings.Repeat("a", domain.MaxSlugLen) // already at the limit, alone
	for suffixLen := 7; suffixLen <= 10; suffixLen++ {
		suffix := strings.Repeat("x", suffixLen)
		candidate, err := buildSuffixedCandidate(base, suffix)
		if err != nil {
			t.Fatalf("suffixLen=%d: unexpected error: %v", suffixLen, err)
		}
		if len(candidate) > domain.MaxSlugLen {
			t.Fatalf("suffixLen=%d: candidate length %d exceeds domain.MaxSlugLen %d (candidate=%q)", suffixLen, len(candidate), domain.MaxSlugLen, candidate)
		}
		if strings.HasPrefix(candidate, "-") {
			t.Fatalf("suffixLen=%d: candidate %q starts with a hyphen — invalid slug format", suffixLen, candidate)
		}
	}
}

// TestBuildSuffixedCandidateErrorsWhenNoRoomForBase guards code-review
// finding #3: an earlier version had no guard here, so once suffix
// escalation left no room for even a 1-character base, it silently produced
// a leading-hyphen candidate ("-"+suffix) that violates the slug format
// (and would have failed the DB CHECK constraint with an unhandled error
// instead of failing cleanly here).
func TestBuildSuffixedCandidateErrorsWhenNoRoomForBase(t *testing.T) {
	suffix := strings.Repeat("x", domain.MaxSlugLen) // consumes the entire budget by itself
	_, err := buildSuffixedCandidate("any-base", suffix)
	if err == nil {
		t.Fatal("expected an error when the suffix alone leaves no room for any base")
	}
}
