package infra

import (
	"testing"

	"mycourse-io-be/internal/taxonomy/domain"
)

type outcomeHydrateCase struct {
	name         string
	locale       string
	rows         map[string]outcomeTranslationRow
	wantShort    string
	wantResolved string
	wantDesc     []string // nil means expect empty description
}

func outcomeHydrateCases() []outcomeHydrateCase {
	return []outcomeHydrateCase{
		{
			name:   "exact_locale_same_row",
			locale: "vi",
			rows: map[string]outcomeTranslationRow{
				"vi": {Locale: "vi", ShortDescription: "Vi short", Description: descriptionJSONB{"vi desc"}},
				"en": {Locale: "en", ShortDescription: "En short", Description: descriptionJSONB{"en desc"}},
			},
			wantShort: "Vi short", wantResolved: "vi", wantDesc: []string{"vi desc"},
		},
		{
			name:   "empty_description_no_en_fallback",
			locale: "vi",
			rows: map[string]outcomeTranslationRow{
				"vi": {Locale: "vi", ShortDescription: "Vi short", Description: descriptionJSONB{}},
				"en": {Locale: "en", ShortDescription: "En short", Description: descriptionJSONB{"en desc"}},
			},
			wantShort: "Vi short", wantResolved: "vi", wantDesc: nil,
		},
		{
			name: "canonical_keeps_description", locale: "fr",
			rows:      map[string]outcomeTranslationRow{},
			wantShort: "Canonical short", wantResolved: "canonical", wantDesc: []string{"canonical desc"},
		},
		{
			name:   "base_locale_row",
			locale: "pt-BR",
			rows: map[string]outcomeTranslationRow{
				"pt": {Locale: "pt", ShortDescription: "Pt short", Description: descriptionJSONB{"pt desc"}},
				"en": {Locale: "en", ShortDescription: "En short", Description: descriptionJSONB{"en desc"}},
			},
			wantShort: "Pt short", wantResolved: "pt", wantDesc: []string{"pt desc"},
		},
	}
}

func TestApplyOutcomeLocaleHydrate(t *testing.T) {
	t.Parallel()
	for _, tc := range outcomeHydrateCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			item := domain.CourseOutcome{
				ShortDescription: "Canonical short",
				Description:      []string{"canonical desc"},
			}
			applyOutcomeLocaleHydrate(&item, tc.rows, tc.locale)
			if item.ShortDescription != tc.wantShort || item.ResolvedLocale != tc.wantResolved {
				t.Fatalf("short/locale=%q/%q want %q/%q",
					item.ShortDescription, item.ResolvedLocale, tc.wantShort, tc.wantResolved)
			}
			if tc.wantDesc == nil {
				if len(item.Description) != 0 {
					t.Fatalf("description=%v want empty", item.Description)
				}
				return
			}
			if len(item.Description) != len(tc.wantDesc) || item.Description[0] != tc.wantDesc[0] {
				t.Fatalf("description=%v want %v", item.Description, tc.wantDesc)
			}
		})
	}
}
