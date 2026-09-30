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

func TestGenerateAutoSlugBaseFromNormalTitle(t *testing.T) {
	base, mustSuffix := generateAutoSlugBase("Introduction to Go")
	if mustSuffix {
		t.Fatalf("mustSuffix = true, want false for a normal English title")
	}
	if base == "" {
		t.Fatal("expected a non-empty base slug")
	}
	if strings.ContainsAny(base, " _") {
		t.Fatalf("base %q contains characters that should have been transliterated away", base)
	}
}

func TestGenerateAutoSlugBaseTransliteratesNonLatinTitle(t *testing.T) {
	// gosimple/slug (backed by gosimple/unidecode) transliterates CJK/Cyrillic/
	// Thai to ASCII rather than passing it through raw or dropping it.
	base, mustSuffix := generateAutoSlugBase("影師")
	if mustSuffix {
		t.Fatalf("mustSuffix = true for a transliterable Chinese title, base=%q", base)
	}
	for _, r := range base {
		if r > 127 {
			t.Fatalf("base %q contains non-ASCII rune %q — transliteration should have produced pure ASCII", base, r)
		}
	}
}

func TestGenerateAutoSlugBaseFallsBackOnEmojiOnlyTitle(t *testing.T) {
	base, mustSuffix := generateAutoSlugBase("🔥🔥🔥")
	if !mustSuffix {
		t.Fatalf("mustSuffix = false, want true for an emoji-only title with nothing to transliterate")
	}
	if base != "course" {
		t.Fatalf("base = %q, want \"course\" fallback", base)
	}
}

func TestGenerateAutoSlugBaseTruncatesLongTitle(t *testing.T) {
	base, _ := generateAutoSlugBase(strings.Repeat("influencer marketing strategy ", 20)) // well over 255 chars once transliterated
	if len(base) > domain.MaxSlugLen {
		t.Fatalf("base length %d exceeds domain.MaxSlugLen %d", len(base), domain.MaxSlugLen)
	}
}

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
