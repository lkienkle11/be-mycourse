package taxonomy

import "testing"

func TestResolveNodeNameFallback(t *testing.T) {
	t.Parallel()
	n := TreeNode{
		Name: "Canonical",
		Translations: map[string]NodeTranslation{
			"en": {Name: "English"},
			"vi": {Name: "Tiếng Việt"},
		},
	}
	got, loc := ResolveNodeName(n, "vi")
	if got != "Tiếng Việt" || loc != "vi" {
		t.Fatalf("got %q %q", got, loc)
	}
	got, loc = ResolveNodeName(n, "fr")
	if got != "English" || loc != "en" {
		t.Fatalf("en fallback got %q %q", got, loc)
	}
}

func TestSyncCanonicalAndEn(t *testing.T) {
	t.Parallel()
	_, _, conflict := SyncCanonicalAndEn("A", map[string]NodeTranslation{"en": {Name: "B"}})
	if !conflict {
		t.Fatal("expected conflict")
	}
	name, tr, conflict := SyncCanonicalAndEn("A", nil)
	if conflict || name != "A" || tr["en"].Name != "A" {
		t.Fatalf("mirror canonical: %q %#v conflict=%v", name, tr, conflict)
	}
	name, tr, conflict = SyncCanonicalAndEn("", map[string]NodeTranslation{"en": {Name: "E"}})
	if conflict || name != "E" || tr["en"].Name != "E" {
		t.Fatalf("mirror en: %q %#v conflict=%v", name, tr, conflict)
	}
}

func TestEnsureEnTranslationIdempotent(t *testing.T) {
	t.Parallel()
	in := []TreeNode{{
		ID:   "1",
		Name: "Root",
		Translations: map[string]NodeTranslation{
			"en": {Name: "Keep"},
		},
		Children: []TreeNode{{ID: "2", Name: "Child"}},
	}}
	out := EnsureEnTranslation(in)
	if out[0].Translations["en"].Name != "Keep" {
		t.Fatalf("must not overwrite en: %#v", out[0].Translations)
	}
	if out[0].Children[0].Translations["en"].Name != "Child" {
		t.Fatalf("child en: %#v", out[0].Children[0].Translations)
	}
}
