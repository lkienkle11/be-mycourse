package i18n

import "testing"

func TestCanonicalizeLocale(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"en", "en", false},
		{"EN", "en", false},
		{"en-us", "en-US", false},
		{"pt-br", "pt-BR", false},
		{"pt-BR", "pt-BR", false},
		{"ja-JP", "ja-JP", false},
		{"zh-hant", "zh-Hant", false},
		{"", "", true},
		{"   ", "", true},
		{"en--US", "", true},
		{"1en", "", true},
		{"abcdefgh", "abcdefgh", false},           // 8-letter primary language
		{"en-Latn-US-x1", "en-Latn-US-x1", false}, // 13 chars
		{"en-Latn-US-extra1", "", true},           // 17 > MaxLocaleLen (16)
	}
	for _, tc := range cases {
		got, err := CanonicalizeLocale(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("CanonicalizeLocale(%q) err=nil want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("CanonicalizeLocale(%q) err=%v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("CanonicalizeLocale(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNegotiateReadLocale(t *testing.T) {
	t.Parallel()
	if got := NegotiateReadLocale(""); got != DefaultLocale {
		t.Fatalf("empty → %q", got)
	}
	if got := NegotiateReadLocale("!!!"); got != DefaultLocale {
		t.Fatalf("invalid → %q", got)
	}
	if got := NegotiateReadLocale("vi"); got != "vi" {
		t.Fatalf("vi → %q", got)
	}
	if got := NegotiateReadLocale("en-us"); got != "en-US" {
		t.Fatalf("en-us → %q", got)
	}
}

func TestLocaleCandidates(t *testing.T) {
	t.Parallel()
	exact, base := LocaleCandidates("")
	if exact != DefaultLocale || base != DefaultLocale {
		t.Fatalf("empty → %q %q", exact, base)
	}
	exact, base = LocaleCandidates("pt-BR")
	if exact != "pt-BR" || base != "pt" {
		t.Fatalf("pt-BR → %q %q", exact, base)
	}
	exact, base = LocaleCandidates("vi")
	if exact != "vi" || base != "vi" {
		t.Fatalf("vi → %q %q", exact, base)
	}
}

func TestResolveText(t *testing.T) {
	t.Parallel()
	tr := map[string]string{
		"vi":    "Tiếng Việt",
		"en":    "English",
		"en-US": "US English",
	}
	text, loc := ResolveText("vi", tr, "Canonical")
	if text != "Tiếng Việt" || loc != "vi" {
		t.Fatalf("exact: %q %q", text, loc)
	}
	text, loc = ResolveText("en-GB", tr, "Canonical")
	if text != "English" || loc != "en" {
		t.Fatalf("base fallback: %q %q", text, loc)
	}
	text, loc = ResolveText("fr", map[string]string{"en": "English"}, "Canonical")
	if text != "English" || loc != "en" {
		t.Fatalf("default en: %q %q", text, loc)
	}
	text, loc = ResolveText("fr", map[string]string{}, "Canonical")
	if text != "Canonical" || loc != "canonical" {
		t.Fatalf("canonical: %q %q", text, loc)
	}
}
