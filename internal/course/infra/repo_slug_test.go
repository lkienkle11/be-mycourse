package infra

import (
	stderrors "errors"
	"strings"
	"testing"
)

func TestCourseSlugCandidate(t *testing.T) {
	tests := []struct {
		base    string
		attempt int
		want    string
	}{
		{"react-basics", 1, "react-basics"},
		{"react-basics", 2, "react-basics-2"},
		{"react-basics", 10, "react-basics-10"},
	}
	for _, tc := range tests {
		if got := courseSlugCandidate(tc.base, tc.attempt); got != tc.want {
			t.Errorf("courseSlugCandidate(%q, %d) = %q, want %q", tc.base, tc.attempt, got, tc.want)
		}
	}
}

func TestCourseSlugCandidateRespectsMaxLength(t *testing.T) {
	longBase := strings.Repeat("a", maxCourseSlugLen)
	got := courseSlugCandidate(longBase, 2)
	if len(got) > maxCourseSlugLen {
		t.Fatalf("slug length %d exceeds max %d", len(got), maxCourseSlugLen)
	}
	if !strings.HasSuffix(got, "-2") {
		t.Fatalf("expected -2 suffix, got %q", got)
	}
}

func TestIsCourseSlugNumericSuffixVariant(t *testing.T) {
	base := "react-basics"
	tests := []struct {
		slug string
		want bool
	}{
		{"react-basics", true},
		{"react-basics-2", true},
		{"react-basics-10", true},
		{"react-basics-advanced", false},
		{"react-basics-2-extra", false},
		{"other", false},
	}
	for _, tc := range tests {
		if got := isCourseSlugNumericSuffixVariant(tc.slug, base); got != tc.want {
			t.Errorf("isCourseSlugNumericSuffixVariant(%q, %q) = %v, want %v", tc.slug, base, got, tc.want)
		}
	}
}

func TestNextFreeCourseSlug(t *testing.T) {
	base := "react-basics"
	if got := nextFreeCourseSlug(base, nil); got != "react-basics" {
		t.Fatalf("empty taken: got %q", got)
	}
	if got := nextFreeCourseSlug(base, map[string]struct{}{base: {}}); got != "react-basics-2" {
		t.Fatalf("base taken: got %q", got)
	}
	taken := map[string]struct{}{
		base:              {},
		"react-basics-2":  {},
		"react-basics-4":  {},
		"react-basics-advanced": {},
	}
	if got := nextFreeCourseSlug(base, taken); got != "react-basics-3" {
		t.Fatalf("fill gap: got %q", got)
	}
}

func TestIsCourseSlugDuplicateKey(t *testing.T) {
	err := stderrors.New(`ERROR: duplicate key value violates unique constraint "uix_courses_slug_active"`)
	if !isCourseSlugDuplicateKey(err) {
		t.Fatal("expected duplicate slug detection")
	}
	if isCourseSlugDuplicateKey(nil) {
		t.Fatal("nil should not match")
	}
}
