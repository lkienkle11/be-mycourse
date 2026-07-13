package infra

import (
	"strings"
	"testing"
)

func TestEscapeSQLStringLiteral(t *testing.T) {
	t.Parallel()
	if got := escapeSQLStringLiteral("en-US"); got != "en-US" {
		t.Fatalf("got %q", got)
	}
	if got := escapeSQLStringLiteral("a'b"); got != "a''b" {
		t.Fatalf("got %q", got)
	}
}

func TestLocalizedResolvedLocaleSelect(t *testing.T) {
	t.Parallel()
	got := localizedResolvedLocaleSelect("pt-BR", "pt")
	for _, frag := range []string{
		"THEN 'pt-BR'",
		"THEN 'pt'",
		"THEN 'en'",
		"ELSE 'canonical'",
		"AS resolved_locale",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in %q", frag, got)
		}
	}
}
