package infra

import (
	"strings"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/i18n"
)

func taxonomyLocaleJoinArgs(requested string) (exact, base string) {
	return i18n.LocaleCandidates(requested)
}

// joinLocalizedTopicName adds translation joins for course_topics alias `ct`.
func joinLocalizedTopicName(qJoins string) string {
	return qJoins +
		" LEFT JOIN " + constants.TableTaxonomyCourseTopicTranslations + " tr_exact ON tr_exact.topic_id = ct.id AND tr_exact.locale = ?" +
		" LEFT JOIN " + constants.TableTaxonomyCourseTopicTranslations + " tr_base ON tr_base.topic_id = ct.id AND tr_base.locale = ?" +
		" LEFT JOIN " + constants.TableTaxonomyCourseTopicTranslations + " tr_en ON tr_en.topic_id = ct.id AND tr_en.locale = '" + i18n.DefaultLocale + "'"
}

func joinLocalizedSkillName(qJoins string) string {
	return qJoins +
		" LEFT JOIN " + constants.TableTaxonomyCourseSkillTranslations + " tr_exact ON tr_exact.skill_id = cs.id AND tr_exact.locale = ?" +
		" LEFT JOIN " + constants.TableTaxonomyCourseSkillTranslations + " tr_base ON tr_base.skill_id = cs.id AND tr_base.locale = ?" +
		" LEFT JOIN " + constants.TableTaxonomyCourseSkillTranslations + " tr_en ON tr_en.skill_id = cs.id AND tr_en.locale = '" + i18n.DefaultLocale + "'"
}

func localizedNameSelect(rootAlias string) string {
	return "COALESCE(NULLIF(tr_exact.name, ''), NULLIF(tr_base.name, ''), NULLIF(tr_en.name, ''), " + rootAlias + ".name, '') AS name"
}

// localizedResolvedLocaleSelect reports which COALESCE branch produced the display name.
// exact/base must already be negotiated BCP47 tags (from LocaleCandidates).
func localizedResolvedLocaleSelect(exact, base string) string {
	return "CASE" +
		" WHEN NULLIF(tr_exact.name, '') IS NOT NULL THEN '" + escapeSQLStringLiteral(exact) + "'" +
		" WHEN NULLIF(tr_base.name, '') IS NOT NULL THEN '" + escapeSQLStringLiteral(base) + "'" +
		" WHEN NULLIF(tr_en.name, '') IS NOT NULL THEN '" + i18n.DefaultLocale + "'" +
		" ELSE 'canonical' END AS resolved_locale"
}

func escapeSQLStringLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
