package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/taxonomy/domain"
)

type nameDetailConfig[T any] struct {
	DB      *gorm.DB
	Table   string
	FKCol   string
	GetByID func(context.Context, string) (*T, error)
	IDOf    func(*T) string
	SetEdit func(*T, map[string]taxpkg.NodeTranslation, []string)
	Hydrate func(context.Context, *gorm.DB, string, []T) error
}

func (c nameDetailConfig[T]) fetch(ctx context.Context, id, locale string, viewEdit bool) (*T, error) {
	row, err := c.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if viewEdit {
		tr, locales, err := loadAllNameTranslations(ctx, c.DB, c.Table, c.FKCol, c.IDOf(row))
		if err != nil {
			return nil, err
		}
		c.SetEdit(row, tr, locales)
		return row, nil
	}
	items := []T{*row}
	if err := c.Hydrate(ctx, c.DB, locale, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *GormCourseTopicRepository) GetDetail(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseTopic, error) {
	cfg := nameDetailConfig[domain.CourseTopic]{
		DB: r.db, Table: constants.TableTaxonomyCourseTopicTranslations, FKCol: "topic_id",
		GetByID: r.GetByID, IDOf: func(t *domain.CourseTopic) string { return t.ID },
		Hydrate: hydrateCourseTopicsLocale,
	}
	cfg.SetEdit = func(t *domain.CourseTopic, tr map[string]taxpkg.NodeTranslation, locales []string) {
		t.Translations, t.AvailableLocales = tr, locales
	}
	return cfg.fetch(ctx, id, locale, viewEdit)
}

func (r *GormCourseOutcomeRepository) GetDetail(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseOutcome, error) {
	row, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !viewEdit {
		items := []domain.CourseOutcome{*row}
		if err := hydrateCourseOutcomesLocale(ctx, r.db, locale, items); err != nil {
			return nil, err
		}
		return &items[0], nil
	}
	tr, locales, err := loadAllOutcomeTranslations(ctx, r.db, row.ID)
	if err != nil {
		return nil, err
	}
	row.Translations = tr
	row.AvailableLocales = locales
	return row, nil
}

func (r *GormCourseSkillRepository) GetDetail(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseSkill, error) {
	return nameDetailConfig[domain.CourseSkill]{
		DB: r.db, Table: constants.TableTaxonomyCourseSkillTranslations, FKCol: "skill_id",
		GetByID: r.GetByID,
		IDOf:    func(s *domain.CourseSkill) string { return s.ID },
		SetEdit: func(s *domain.CourseSkill, tr map[string]taxpkg.NodeTranslation, locales []string) {
			s.Translations = tr
			s.AvailableLocales = locales
		},
		Hydrate: hydrateCourseSkillsLocale,
	}.fetch(ctx, id, locale, viewEdit)
}

func (r *GormTagRepository) GetDetail(ctx context.Context, id, locale string, viewEdit bool) (*domain.Tag, error) {
	cfg := nameDetailConfig[domain.Tag]{
		DB: r.db, Table: constants.TableTaxonomyTagTranslations, FKCol: "tag_id",
		GetByID: r.GetByID, IDOf: func(t *domain.Tag) string { return t.ID },
		Hydrate: hydrateTagsLocale,
		SetEdit: func(t *domain.Tag, tr map[string]taxpkg.NodeTranslation, locales []string) {
			t.Translations, t.AvailableLocales = tr, locales
		},
	}
	return cfg.fetch(ctx, id, locale, viewEdit)
}

func (r *GormCourseLevelRepository) GetDetail(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseLevel, error) {
	row, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if viewEdit {
		tr, locales, err := loadAllNameTranslations(ctx, r.db, constants.TableTaxonomyCourseLevelTranslations, "course_level_id", row.ID)
		if err != nil {
			return nil, err
		}
		row.Translations = tr
		row.AvailableLocales = locales
		return row, nil
	}
	items := []domain.CourseLevel{*row}
	if err := hydrateCourseLevelsLocale(ctx, r.db, locale, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}
