package slug

import (
	"strings"
	"testing"
)

func TestGenerateAutoSlugBaseFromNormalName(t *testing.T) {
	base, mustSuffix := GenerateAutoSlugBase("Introduction to Go", 255, KindCourse)
	if mustSuffix {
		t.Fatalf("mustSuffix = true, want false for a normal English name")
	}
	if base == "" {
		t.Fatal("expected a non-empty base slug")
	}
	if strings.ContainsAny(base, " _") {
		t.Fatalf("base %q contains characters that should have been transliterated away", base)
	}
}

func TestGenerateAutoSlugBaseTransliteratesNonLatinName(t *testing.T) {
	// gosimple/slug (backed by gosimple/unidecode) transliterates CJK/
	// Vietnamese/Cyrillic/Thai to ASCII rather than passing it through raw
	// or dropping it, for every kind (the mechanism is kind-independent).
	for _, name := range []string{"影師", "Lập trình Golang"} {
		base, mustSuffix := GenerateAutoSlugBase(name, 255, KindCourseTopic)
		if mustSuffix {
			t.Fatalf("mustSuffix = true for transliterable name %q, base=%q", name, base)
		}
		for _, r := range base {
			if r > 127 {
				t.Fatalf("base %q contains non-ASCII rune %q — transliteration should have produced pure ASCII", base, r)
			}
		}
	}
}

func TestGenerateAutoSlugBaseFallsBackOnEmojiOnlyNamePerKind(t *testing.T) {
	cases := []struct {
		kind Kind
		want string
	}{
		{KindCourse, "course"},
		{KindCourseLevel, "level"},
		{KindCourseTopic, "topic"},
		{KindCourseSkill, "skill"},
		{KindTag, "tag"},
		{KindTreeNode, "node"},
	}
	for _, tc := range cases {
		base, mustSuffix := GenerateAutoSlugBase("🔥🔥🔥", 255, tc.kind)
		if !mustSuffix {
			t.Fatalf("kind=%s: mustSuffix = false, want true for an emoji-only name with nothing to transliterate", tc.kind)
		}
		if base != tc.want {
			t.Fatalf("kind=%s: base = %q, want %q fallback", tc.kind, base, tc.want)
		}
	}
}

func TestGenerateAutoSlugBaseTruncatesLongName(t *testing.T) {
	const maxLen = 255
	base, _ := GenerateAutoSlugBase(strings.Repeat("influencer marketing strategy ", 20), maxLen, KindCourseSkill)
	if len(base) > maxLen {
		t.Fatalf("base length %d exceeds maxLen %d", len(base), maxLen)
	}
}
