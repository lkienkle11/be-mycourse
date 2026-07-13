package infra

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/i18n"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/uuidx"
	"mycourse-io-be/internal/taxonomy/domain"
)

func loadAllNameTranslations(
	ctx context.Context,
	db *gorm.DB,
	table, fkCol, parentID string,
) (map[string]taxpkg.NodeTranslation, []string, error) {
	var rows []nameTranslationRow
	err := db.WithContext(ctx).Table(table).
		Select(fkCol+" AS parent_id, locale, name").
		Where(fkCol+" = ?", parentID).
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	out := make(map[string]taxpkg.NodeTranslation, len(rows))
	locales := make([]string, 0, len(rows))
	seen := map[string]struct{}{}
	for _, r := range rows {
		loc := strings.TrimSpace(r.Locale)
		out[loc] = taxpkg.NodeTranslation{Name: r.Name}
		if _, ok := seen[loc]; ok {
			continue
		}
		seen[loc] = struct{}{}
		locales = append(locales, loc)
	}
	if _, ok := seen[i18n.DefaultLocale]; !ok {
		locales = append([]string{i18n.DefaultLocale}, locales...)
	}
	return out, locales, nil
}

func upsertNameTranslations(
	ctx context.Context,
	tx *gorm.DB,
	table, fkCol, parentID string,
	translations map[string]taxpkg.NodeTranslation,
) error {
	keep := make([]string, 0, len(translations))
	now := timex.NowUnix()
	for locale, nt := range translations {
		name := strings.TrimSpace(nt.Name)
		loc := strings.TrimSpace(locale)
		if name == "" || loc == "" {
			continue
		}
		id, err := uuidx.NewV7()
		if err != nil {
			return err
		}
		row := map[string]any{
			"id":         id,
			fkCol:        parentID,
			"locale":     loc,
			"name":       name,
			"created_at": now,
			"updated_at": now,
		}
		err = tx.WithContext(ctx).Table(table).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: fkCol}, {Name: "locale"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at"}),
		}).Create(row).Error
		if err != nil {
			return err
		}
		keep = append(keep, loc)
	}
	return deleteMissingTranslations(ctx, tx, table, fkCol, parentID, keep)
}

func loadAllOutcomeTranslations(
	ctx context.Context,
	db *gorm.DB,
	parentID string,
) (map[string]domain.OutcomeTranslation, []string, error) {
	var rows []outcomeTranslationRow
	err := db.WithContext(ctx).Table(constants.TableTaxonomyCourseOutcomeTranslations).
		Select("outcome_id AS parent_id, locale, short_description, description").
		Where("outcome_id = ?", parentID).
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	out := make(map[string]domain.OutcomeTranslation, len(rows))
	locales := make([]string, 0, len(rows))
	seen := map[string]struct{}{}
	for _, r := range rows {
		loc := strings.TrimSpace(r.Locale)
		desc := []string(r.Description)
		if desc == nil {
			desc = []string{}
		}
		out[loc] = domain.OutcomeTranslation{
			ShortDescription: r.ShortDescription,
			Description:      desc,
		}
		if _, ok := seen[loc]; ok {
			continue
		}
		seen[loc] = struct{}{}
		locales = append(locales, loc)
	}
	if _, ok := seen[i18n.DefaultLocale]; !ok {
		locales = append([]string{i18n.DefaultLocale}, locales...)
	}
	return out, locales, nil
}

func upsertOutcomeTranslations(
	ctx context.Context,
	tx *gorm.DB,
	parentID string,
	translations map[string]domain.OutcomeTranslation,
) error {
	keep := make([]string, 0, len(translations))
	now := timex.NowUnix()
	for locale, ot := range translations {
		short := strings.TrimSpace(ot.ShortDescription)
		loc := strings.TrimSpace(locale)
		if short == "" || loc == "" {
			continue
		}
		desc := ot.Description
		if desc == nil {
			desc = []string{}
		}
		descVal, err := descriptionJSONB(desc).Value()
		if err != nil {
			return err
		}
		id, err := uuidx.NewV7()
		if err != nil {
			return err
		}
		row := map[string]any{
			"id":                id,
			"outcome_id":        parentID,
			"locale":            loc,
			"short_description": short,
			"description":       descVal,
			"created_at":        now,
			"updated_at":        now,
		}
		err = tx.WithContext(ctx).Table(constants.TableTaxonomyCourseOutcomeTranslations).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "outcome_id"}, {Name: "locale"}},
			DoUpdates: clause.AssignmentColumns([]string{"short_description", "description", "updated_at"}),
		}).Create(row).Error
		if err != nil {
			return err
		}
		keep = append(keep, loc)
	}
	return deleteMissingTranslations(ctx, tx, constants.TableTaxonomyCourseOutcomeTranslations, "outcome_id", parentID, keep)
}

func deleteMissingTranslations(
	ctx context.Context,
	tx *gorm.DB,
	table, fkCol, parentID string,
	keep []string,
) error {
	q := tx.WithContext(ctx).Table(table).Where(fkCol+" = ?", parentID)
	if len(keep) == 0 {
		return q.Where("1 = 1").Delete(map[string]any{}).Error
	}
	return q.Where("locale NOT IN ?", keep).Delete(map[string]any{}).Error
}
