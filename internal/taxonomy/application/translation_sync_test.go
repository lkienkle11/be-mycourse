package application

import (
	"errors"
	"strings"
	"testing"

	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/taxonomy/domain"
)

func TestCanonicalizeNameTranslations_CollisionDifferentPayload(t *testing.T) {
	t.Parallel()
	_, err := canonicalizeNameTranslations(map[string]taxpkg.NodeTranslation{
		"en-us": {Name: "A"},
		"en-US": {Name: "B"},
	})
	if err == nil || !errors.Is(err, ErrTaxonomyValidation) {
		t.Fatalf("want ErrTaxonomyValidation, got %v", err)
	}
}

func TestCanonicalizeNameTranslations_CollisionIdenticalPayload(t *testing.T) {
	t.Parallel()
	out, err := canonicalizeNameTranslations(map[string]taxpkg.NodeTranslation{
		"en-us": {Name: "Same"},
		"en-US": {Name: "Same"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out["en-US"].Name != "Same" {
		t.Fatalf("got %#v", out)
	}
}

func TestCanonicalizeOutcomeTranslations_RequiresShortWhenDescriptionPresent(t *testing.T) {
	t.Parallel()
	_, err := canonicalizeOutcomeTranslations(map[string]domain.OutcomeTranslation{
		"vi": {ShortDescription: "", Description: []string{"line"}},
	})
	if err == nil || !errors.Is(err, ErrTaxonomyValidation) {
		t.Fatalf("want ErrTaxonomyValidation, got %v", err)
	}
}

func TestCanonicalizeOutcomeTranslations_DropsEmpty(t *testing.T) {
	t.Parallel()
	out, err := canonicalizeOutcomeTranslations(map[string]domain.OutcomeTranslation{
		"vi": {ShortDescription: "  ", Description: []string{"", "  "}},
		"fr": {ShortDescription: "Bonjour", Description: []string{"a"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["vi"]; ok {
		t.Fatalf("empty vi should be dropped: %#v", out)
	}
	if out["fr"].ShortDescription != "Bonjour" {
		t.Fatalf("fr short: %#v", out["fr"])
	}
}

func TestCanonicalizeNameTranslations_RejectsLongName(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", taxpkg.DefaultMaxNameLen+1)
	_, err := canonicalizeNameTranslations(map[string]taxpkg.NodeTranslation{
		"vi": {Name: long},
	})
	if err == nil || !errors.Is(err, ErrTaxonomyValidation) {
		t.Fatalf("want ErrTaxonomyValidation, got %v", err)
	}
}

func TestCanonicalizeOutcomeTranslations_RejectsLongShort(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("y", 101)
	_, err := canonicalizeOutcomeTranslations(map[string]domain.OutcomeTranslation{
		"pt-BR": {ShortDescription: long, Description: []string{}},
	})
	if err == nil || !errors.Is(err, ErrTaxonomyValidation) {
		t.Fatalf("want ErrTaxonomyValidation, got %v", err)
	}
}

func TestCanonicalizeOutcomeTranslations_RejectsTooManyParagraphs(t *testing.T) {
	t.Parallel()
	paras := make([]string, taxpkg.DefaultMaxDescriptionItems+1)
	for i := range paras {
		paras[i] = "line"
	}
	_, err := canonicalizeOutcomeTranslations(map[string]domain.OutcomeTranslation{
		"fr": {ShortDescription: "ok", Description: paras},
	})
	if err == nil || !errors.Is(err, ErrTaxonomyValidation) {
		t.Fatalf("want ErrTaxonomyValidation, got %v", err)
	}
}

func TestReplaceNameTranslations_UsesSubmittedMapOnly(t *testing.T) {
	t.Parallel()
	patch := map[string]taxpkg.NodeTranslation{
		"en": {Name: "English"},
		"vi": {Name: "new-vi"},
	}
	got := replaceNameTranslations(patch)
	if _, ok := got["fr"]; ok {
		t.Fatalf("fr should be removed on full replace: %#v", got)
	}
	if got["vi"].Name != "new-vi" || got["en"].Name != "English" {
		t.Fatalf("unexpected replace result: %#v", got)
	}
}

func TestDeleteMissingKeepLocalesLogic(t *testing.T) {
	t.Parallel()
	// Document expected keep-set behavior used by upsert*Translations:
	// locales with empty content are omitted from keep and therefore deleted.
	translations := map[string]taxpkg.NodeTranslation{
		"en": {Name: "English"},
		"vi": {Name: ""},
	}
	keep := make([]string, 0, len(translations))
	for loc, nt := range translations {
		if nt.Name == "" {
			continue
		}
		keep = append(keep, loc)
	}
	if len(keep) != 1 || keep[0] != "en" {
		t.Fatalf("keep=%v want [en]", keep)
	}
}
