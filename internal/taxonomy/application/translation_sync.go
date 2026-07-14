package application

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"mycourse-io-be/internal/shared/i18n"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/taxonomy/domain"
)

func canonicalizeNameTranslations(in map[string]taxpkg.NodeTranslation) (map[string]taxpkg.NodeTranslation, error) {
	if len(in) == 0 {
		return map[string]taxpkg.NodeTranslation{}, nil
	}
	out := make(map[string]taxpkg.NodeTranslation, len(in))
	for raw, nt := range in {
		loc, err := i18n.CanonicalizeLocale(raw)
		if err != nil {
			return nil, errors.Join(ErrTaxonomyValidation, err)
		}
		next := taxpkg.NodeTranslation{Name: strings.TrimSpace(nt.Name)}
		if prev, ok := out[loc]; ok {
			if prev.Name != next.Name {
				return nil, errors.Join(ErrTaxonomyValidation, fmt.Errorf(
					"duplicate translation locale keys collide after canonicalize to %q", loc,
				))
			}
			continue
		}
		out[loc] = next
	}
	if err := validateNameTranslationContents(out); err != nil {
		return nil, err
	}
	return out, nil
}

func validateNameTranslationContents(tr map[string]taxpkg.NodeTranslation) error {
	for loc, nt := range tr {
		name := strings.TrimSpace(nt.Name)
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > taxpkg.DefaultMaxNameLen {
			return errors.Join(ErrTaxonomyValidation, fmt.Errorf(
				"translation name for locale %q must be 1-%d characters", loc, taxpkg.DefaultMaxNameLen,
			))
		}
	}
	return nil
}

func canonicalizeOutcomeTranslations(in map[string]domain.OutcomeTranslation) (map[string]domain.OutcomeTranslation, error) {
	if len(in) == 0 {
		return map[string]domain.OutcomeTranslation{}, nil
	}
	out := make(map[string]domain.OutcomeTranslation, len(in))
	for raw, ot := range in {
		loc, err := i18n.CanonicalizeLocale(raw)
		if err != nil {
			return nil, errors.Join(ErrTaxonomyValidation, err)
		}
		desc := ot.Description
		if desc == nil {
			desc = []string{}
		}
		short := strings.TrimSpace(ot.ShortDescription)
		hasDesc := outcomeDescriptionHasContent(desc)
		if hasDesc && short == "" {
			return nil, errors.Join(ErrTaxonomyValidation, errors.New("short_description is required when description is present for locale "+loc))
		}
		if short == "" && !hasDesc {
			continue
		}
		next := domain.OutcomeTranslation{
			ShortDescription: short,
			Description:      desc,
		}
		if prev, ok := out[loc]; ok {
			if prev.ShortDescription != next.ShortDescription || !stringSlicesEqual(prev.Description, next.Description) {
				return nil, errors.Join(ErrTaxonomyValidation, fmt.Errorf(
					"duplicate translation locale keys collide after canonicalize to %q", loc,
				))
			}
			continue
		}
		out[loc] = next
	}
	if err := validateOutcomeTranslationContents(out); err != nil {
		return nil, err
	}
	return out, nil
}

func validateOutcomeTranslationContents(tr map[string]domain.OutcomeTranslation) error {
	for loc, ot := range tr {
		short := strings.TrimSpace(ot.ShortDescription)
		if short == "" {
			continue
		}
		if utf8.RuneCountInString(short) > 100 {
			return errors.Join(ErrTaxonomyValidation, fmt.Errorf(
				"short_description for locale %q must be 1-100 characters", loc,
			))
		}
		desc := ot.Description
		if desc == nil {
			desc = []string{}
		}
		if err := taxpkg.ValidateDescriptionParagraphs(desc, taxpkg.DefaultMaxDescriptionItems, taxpkg.DefaultMaxDescriptionLen); err != nil {
			return errors.Join(ErrTaxonomyValidation, fmt.Errorf("description for locale %q: %w", loc, err))
		}
	}
	return nil
}

func outcomeDescriptionHasContent(desc []string) bool {
	for _, line := range desc {
		if strings.TrimSpace(line) != "" {
			return true
		}
	}
	return false
}

// replaceNameTranslations treats the submitted patch as the full translation map (not a merge).
func replaceNameTranslations(patch map[string]taxpkg.NodeTranslation) map[string]taxpkg.NodeTranslation {
	out := make(map[string]taxpkg.NodeTranslation, len(patch))
	for k, v := range patch {
		out[k] = v
	}
	return out
}

// replaceOutcomeTranslations treats the submitted patch as the full translation map (not a merge).
func replaceOutcomeTranslations(patch map[string]domain.OutcomeTranslation) map[string]domain.OutcomeTranslation {
	out := make(map[string]domain.OutcomeTranslation, len(patch))
	for k, v := range patch {
		out[k] = v
	}
	return out
}

func syncNameCanonicalAndTranslations(
	canonical string,
	translations map[string]taxpkg.NodeTranslation,
) (name string, out map[string]taxpkg.NodeTranslation, err error) {
	tr, err := canonicalizeNameTranslations(translations)
	if err != nil {
		return "", nil, err
	}
	name, out, conflict := taxpkg.SyncCanonicalAndEn(canonical, tr)
	if conflict {
		return "", nil, domain.ErrTaxonomyCanonicalConflict
	}
	if strings.TrimSpace(name) == "" {
		return "", nil, errors.Join(ErrTaxonomyValidation, errors.New("name is required"))
	}
	return name, out, nil
}

func syncOutcomeCanonicalAndTranslations(
	short string,
	desc []string,
	translations map[string]domain.OutcomeTranslation,
) (string, []string, map[string]domain.OutcomeTranslation, error) {
	tr, err := canonicalizeOutcomeTranslations(translations)
	if err != nil {
		return "", nil, nil, err
	}
	en := tr[i18n.DefaultLocale]
	canonShort, err := syncOutcomeShort(&en, short)
	if err != nil {
		return "", nil, nil, err
	}
	canonDesc, err := syncOutcomeDescription(&en, desc)
	if err != nil {
		return "", nil, nil, err
	}
	tr[i18n.DefaultLocale] = en
	if strings.TrimSpace(canonShort) == "" {
		return "", nil, nil, errors.Join(ErrTaxonomyValidation, errors.New("short_description is required"))
	}
	return canonShort, canonDesc, tr, nil
}

func syncOutcomeShort(en *domain.OutcomeTranslation, short string) (string, error) {
	canonShort := strings.TrimSpace(short)
	enShort := strings.TrimSpace(en.ShortDescription)
	switch {
	case canonShort != "" && enShort != "" && canonShort != enShort:
		return "", domain.ErrTaxonomyCanonicalConflict
	case canonShort != "" && enShort == "":
		en.ShortDescription = canonShort
	case canonShort == "" && enShort != "":
		canonShort = enShort
		en.ShortDescription = enShort
	default:
		if canonShort != "" {
			en.ShortDescription = canonShort
		}
	}
	return canonShort, nil
}

func syncOutcomeDescription(en *domain.OutcomeTranslation, desc []string) ([]string, error) {
	canonDesc := desc
	if canonDesc == nil {
		canonDesc = []string{}
	}
	enDesc := en.Description
	if enDesc == nil {
		enDesc = []string{}
	}
	switch {
	case len(canonDesc) > 0 && len(enDesc) > 0 && !stringSlicesEqual(canonDesc, enDesc):
		return nil, domain.ErrTaxonomyCanonicalConflict
	case len(canonDesc) > 0 && len(enDesc) == 0:
		enDesc = append([]string{}, canonDesc...)
	case len(canonDesc) == 0 && len(enDesc) > 0:
		canonDesc = append([]string{}, enDesc...)
	default:
		if len(canonDesc) > 0 {
			enDesc = append([]string{}, canonDesc...)
		}
	}
	en.Description = enDesc
	return canonDesc, nil
}

func prepareTreeWrite(nodes []taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	if nodes == nil {
		return nil, nil
	}
	canonized, err := canonicalizeTreeLocales(nodes)
	if err != nil {
		return nil, err
	}
	out, conflict := taxpkg.PrepareTreeForWrite(canonized)
	if conflict {
		return nil, domain.ErrTaxonomyCanonicalConflict
	}
	return out, nil
}

func canonicalizeTreeLocales(nodes []taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}
	out := make([]taxpkg.TreeNode, len(nodes))
	for i, n := range nodes {
		tr, err := canonicalizeNameTranslations(n.Translations)
		if err != nil {
			return nil, err
		}
		children, err := canonicalizeTreeLocales(n.Children)
		if err != nil {
			return nil, err
		}
		out[i] = taxpkg.TreeNode{
			ID:           n.ID,
			Name:         n.Name,
			Slug:         n.Slug,
			Translations: tr,
			Children:     children,
		}
	}
	return out, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}
