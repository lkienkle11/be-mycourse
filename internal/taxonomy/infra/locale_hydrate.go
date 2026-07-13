package infra

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/i18n"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/taxonomy/domain"
)

type nameTranslationRow struct {
	ParentID string `gorm:"column:parent_id"`
	Locale   string `gorm:"column:locale"`
	Name     string `gorm:"column:name"`
}

type outcomeTranslationRow struct {
	ParentID         string           `gorm:"column:parent_id"`
	Locale           string           `gorm:"column:locale"`
	ShortDescription string           `gorm:"column:short_description"`
	Description      descriptionJSONB `gorm:"column:description"`
}

func localeCandidates(requested string) (exact, base string) {
	return i18n.LocaleCandidates(requested)
}

func loadNameTranslationMaps(
	ctx context.Context,
	db *gorm.DB,
	table, fkCol string,
	ids []string,
	locales []string,
) (map[string]map[string]string, error) {
	out := make(map[string]map[string]string, len(ids))
	if len(ids) == 0 || len(locales) == 0 {
		return out, nil
	}
	var rows []nameTranslationRow
	err := db.WithContext(ctx).Table(table).
		Select(fkCol+" AS parent_id, locale, name").
		Where(fkCol+" IN ? AND locale IN ?", ids, locales).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		m := out[r.ParentID]
		if m == nil {
			m = map[string]string{}
			out[r.ParentID] = m
		}
		m[r.Locale] = r.Name
	}
	return out, nil
}

func loadOutcomeTranslationMaps(
	ctx context.Context,
	db *gorm.DB,
	ids []string,
	locales []string,
) (map[string]map[string]outcomeTranslationRow, error) {
	out := make(map[string]map[string]outcomeTranslationRow, len(ids))
	if len(ids) == 0 || len(locales) == 0 {
		return out, nil
	}
	var rows []outcomeTranslationRow
	err := db.WithContext(ctx).Table(constants.TableTaxonomyCourseOutcomeTranslations).
		Select("outcome_id AS parent_id, locale, short_description, description").
		Where("outcome_id IN ? AND locale IN ?", ids, locales).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		m := out[r.ParentID]
		if m == nil {
			m = map[string]outcomeTranslationRow{}
			out[r.ParentID] = m
		}
		m[r.Locale] = r
	}
	return out, nil
}

func uniqueLocales(exact, base string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, loc := range []string{exact, base, i18n.DefaultLocale} {
		loc = strings.TrimSpace(loc)
		if loc == "" {
			continue
		}
		if _, ok := seen[loc]; ok {
			continue
		}
		seen[loc] = struct{}{}
		out = append(out, loc)
	}
	return out
}

func applyResolvedName(canonical string, tr map[string]string, requested string) (name, resolved string) {
	return i18n.ResolveText(requested, tr, canonical)
}

type namedLocaleHydrate[T any] struct {
	Table, FKCol string
	IDOf         func(*T) string
	GetName      func(*T) string
	SetResolved  func(*T, string, string)
}

func hydrateNamedLocale[T any](ctx context.Context, db *gorm.DB, locale string, items []T, cfg namedLocaleHydrate[T]) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = cfg.IDOf(&items[i])
	}
	exact, base := localeCandidates(locale)
	trMap, err := loadNameTranslationMaps(ctx, db, cfg.Table, cfg.FKCol, ids, uniqueLocales(exact, base))
	if err != nil {
		return err
	}
	for i := range items {
		name, resolved := applyResolvedName(cfg.GetName(&items[i]), trMap[cfg.IDOf(&items[i])], locale)
		cfg.SetResolved(&items[i], name, resolved)
	}
	return nil
}

func hydrateCourseTopicsLocale(ctx context.Context, db *gorm.DB, locale string, items []domain.CourseTopic) error {
	return hydrateNamedLocale(ctx, db, locale, items, namedLocaleHydrate[domain.CourseTopic]{
		Table: constants.TableTaxonomyCourseTopicTranslations, FKCol: "topic_id",
		IDOf:    func(t *domain.CourseTopic) string { return t.ID },
		GetName: func(t *domain.CourseTopic) string { return t.Name },
		SetResolved: func(t *domain.CourseTopic, name, resolved string) {
			t.Name = name
			t.ResolvedLocale = resolved
			t.ChildTopics = taxpkg.ResolveTreeNames(t.ChildTopics, locale)
		},
	})
}

func hydrateCourseSkillsLocale(ctx context.Context, db *gorm.DB, locale string, items []domain.CourseSkill) error {
	cfg := namedLocaleHydrate[domain.CourseSkill]{
		Table: constants.TableTaxonomyCourseSkillTranslations, FKCol: "skill_id",
		IDOf:    func(s *domain.CourseSkill) string { return s.ID },
		GetName: func(s *domain.CourseSkill) string { return s.Name },
	}
	cfg.SetResolved = func(s *domain.CourseSkill, name, resolved string) {
		s.Name = name
		s.ResolvedLocale = resolved
		s.Children = taxpkg.ResolveTreeNames(s.Children, locale)
	}
	return hydrateNamedLocale(ctx, db, locale, items, cfg)
}

func hydrateTagsLocale(ctx context.Context, db *gorm.DB, locale string, items []domain.Tag) error {
	return hydrateNamedLocale(ctx, db, locale, items, namedLocaleHydrate[domain.Tag]{
		Table: constants.TableTaxonomyTagTranslations, FKCol: "tag_id",
		IDOf:    func(t *domain.Tag) string { return t.ID },
		GetName: func(t *domain.Tag) string { return t.Name },
		SetResolved: func(t *domain.Tag, name, resolved string) {
			t.Name, t.ResolvedLocale = name, resolved
		},
	})
}

func hydrateCourseLevelsLocale(ctx context.Context, db *gorm.DB, locale string, items []domain.CourseLevel) error {
	cfg := namedLocaleHydrate[domain.CourseLevel]{
		Table: constants.TableTaxonomyCourseLevelTranslations, FKCol: "course_level_id",
		IDOf:    func(cl *domain.CourseLevel) string { return cl.ID },
		GetName: func(cl *domain.CourseLevel) string { return cl.Name },
	}
	cfg.SetResolved = func(cl *domain.CourseLevel, name, resolved string) {
		cl.Name = name
		cl.ResolvedLocale = resolved
	}
	return hydrateNamedLocale(ctx, db, locale, items, cfg)
}

func hydrateCourseOutcomesLocale(ctx context.Context, db *gorm.DB, locale string, items []domain.CourseOutcome) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	exact, base := localeCandidates(locale)
	trMap, err := loadOutcomeTranslationMaps(ctx, db, ids, uniqueLocales(exact, base))
	if err != nil {
		return err
	}
	for i := range items {
		applyOutcomeLocaleHydrate(&items[i], trMap[items[i].ID], locale)
	}
	return nil
}

// applyOutcomeLocaleHydrate resolves short_description via ResolveText, then binds
// description from the same translation row only (no independent en description fallback).
func applyOutcomeLocaleHydrate(item *domain.CourseOutcome, rows map[string]outcomeTranslationRow, locale string) {
	shortMap := map[string]string{}
	for loc, row := range rows {
		shortMap[loc] = row.ShortDescription
	}
	short, resolved := i18n.ResolveText(locale, shortMap, item.ShortDescription)
	item.ShortDescription = short
	item.ResolvedLocale = resolved
	if resolved == "canonical" {
		return
	}
	if row, ok := rows[resolved]; ok {
		item.Description = []string(row.Description)
	}
}
