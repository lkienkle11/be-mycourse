package appcli

import (
	"fmt"
	"strings"

	"mycourse-io-be/internal/shared/uuidx"
)

var legacyTableRewriters = map[string]legacyRewriteHandler{
	"user_roles":                     rewriteUserRefOnly,
	"user_permissions":               rewriteUserRefOnly,
	"course_levels":                  rewriteTaxonomyCreatorRows,
	"course_topics":                  rewriteTaxonomyCreatorRows,
	"tags":                           rewriteTaxonomyCreatorRows,
	"course_outcomes":                rewriteTaxonomyCreatorRows,
	"course_skills":                  rewriteTaxonomyCreatorRows,
	"instructor_applications":        rewriteInstructorRows,
	"instructor_profiles":            rewriteInstructorRows,
	"instructor_expertise_topics":    rewriteInstructorRows,
	"instructor_expertise_skills":    rewriteInstructorRows,
	"instructor_tickets":             rewriteInstructorRows,
	"instructor_ticket_messages":     rewriteInstructorTicketMessages,
	"courses":                        rewriteCourses,
	"course_versions":                rewriteCourseVersions,
	"course_version_tags":            rewriteCourseVersionTags,
	"course_version_skills":          rewriteCourseVersionSkills,
	"course_version_outcomes":        rewriteCourseVersionOutcomes,
	"course_collaborators":           rewriteCourseCollaborators,
	"course_sections":                rewriteCourseSections,
	"course_lessons":                 rewriteCourseLessons,
	"course_sub_lessons":             rewriteCourseSubLessons,
	"course_sub_lesson_videos":       rewriteCourseSubLessonVideos,
	"course_sub_lesson_texts":        rewriteCourseSubLessonTextOrQuiz,
	"course_sub_lesson_quizzes":      rewriteCourseSubLessonTextOrQuiz,
	"course_sub_lesson_quiz_options": rewriteCourseSubLessonQuizOptions,
	"course_edit_leases":             rewriteCourseEditLeases,
	"course_enrollments":             rewriteCourseEnrollments,
	"course_progress_items":          rewriteCourseProgressItems,
}

func rewriteLegacyRowForUUID(table string, row map[string]any, idmap *legacyIDMap, coursePatches *[]legacyCourseVersionPatch) (map[string]any, string, string, bool, error) {
	out := make(map[string]any, len(row))
	for k, v := range row {
		out[k] = v
	}

	rc := legacyRewriteCtx{
		table:         table,
		row:           out,
		idmap:         idmap,
		coursePatches: coursePatches,
	}

	var oldID, newID string
	if v, ok := rc.row["id"]; ok && v != nil {
		// Keep RBAC catalog numeric ids unchanged.
		if table != "roles" && table != "media_pending_cloud_cleanup" && table != "system_app_config" {
			oldID = fmt.Sprintf("%v", v)
			gen, err := uuidx.NewV7()
			if err != nil {
				return nil, "", "", false, err
			}
			rc.row["id"] = gen
			newID = gen
		}
	}
	if table == "users" {
		rc.row["user_code"] = uuidx.NewULID()
	}

	if handler, ok := legacyTableRewriters[table]; ok {
		if err := handler(&rc); err != nil {
			return nil, "", "", false, err
		}
	}

	return rc.row, oldID, newID, false, nil
}

func rewriteUserRefOnly(rc *legacyRewriteCtx) error {
	return rc.remapRequired("user_id", "users")
}

func rewriteTaxonomyCreatorRows(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("created_by", "users"); err != nil {
		return err
	}
	return rc.remapOptional("image_file_id", "media_files")
}

func rewriteInstructorRows(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("user_id", "users"); err != nil {
		return err
	}
	if err := rc.remapOptional("topic_id", "course_topics"); err != nil {
		return err
	}
	return rc.remapOptional("skill_id", "course_skills")
}

func rewriteInstructorTicketMessages(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("ticket_id", "instructor_tickets"); err != nil {
		return err
	}
	return rc.remapRequired("author_user_id", "users")
}

func rewriteCourses(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("owner_user_id", "users"); err != nil {
		return err
	}
	patch := legacyCourseVersionPatch{courseNewID: fmt.Sprintf("%v", rc.row["id"])}
	if v, ok := rc.row["current_draft_version_id"]; ok && !isOptionalCol(rc.row, "current_draft_version_id") {
		patch.draftOldID = fmt.Sprintf("%v", v)
		rc.row["current_draft_version_id"] = nil
	}
	if v, ok := rc.row["current_published_version_id"]; ok && !isOptionalCol(rc.row, "current_published_version_id") {
		patch.publishedOldID = fmt.Sprintf("%v", v)
		rc.row["current_published_version_id"] = nil
	}
	if rc.coursePatches != nil && (patch.draftOldID != "" || patch.publishedOldID != "") {
		*rc.coursePatches = append(*rc.coursePatches, patch)
	}
	return nil
}

func rewriteCourseVersions(rc *legacyRewriteCtx) error {
	for _, pair := range [][2]string{
		{"course_id", "courses"},
		{"based_on_version_id", "course_versions"},
		{"thumbnail_file_id", "media_files"},
		{"preview_video_file_id", "media_files"},
		{"course_level_id", "course_levels"},
		{"course_topic_id", "course_topics"},
		{"submitted_by_user_id", "users"},
		{"approved_by_user_id", "users"},
		{"rejected_by_user_id", "users"},
	} {
		if err := rc.remapOptional(pair[0], pair[1]); err != nil {
			return err
		}
	}
	return nil
}

func rewriteCourseVersionTags(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("course_version_id", "course_versions"); err != nil {
		return err
	}
	return rc.remapRequired("tag_id", "tags")
}

func rewriteCourseVersionSkills(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("course_version_id", "course_versions"); err != nil {
		return err
	}
	return rc.remapRequired("skill_id", "course_skills")
}

func rewriteCourseVersionOutcomes(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("course_version_id", "course_versions"); err != nil {
		return err
	}
	return rc.remapRequired("outcome_id", "course_outcomes")
}

func rewriteCourseCollaborators(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("course_id", "courses"); err != nil {
		return err
	}
	return rc.remapRequired("user_id", "users")
}

func rewriteCourseSections(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("course_version_id", "course_versions"); err != nil {
		return err
	}
	if v, ok := rc.row["stable_id"]; ok && fmt.Sprintf("%v", v) != "" {
		rc.row["stable_id"] = mustNewV7()
	}
	return nil
}

func rewriteCourseLessons(rc *legacyRewriteCtx) error {
	return rc.remapCourseLessonLike("section_id", "course_sections")
}

func rewriteCourseSubLessons(rc *legacyRewriteCtx) error {
	return rc.remapCourseLessonLike("lesson_id", "course_lessons")
}

func rewriteCourseSubLessonVideos(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("sub_lesson_id", "course_sub_lessons"); err != nil {
		return err
	}
	return rc.remapRequired("media_file_id", "media_files")
}

func rewriteCourseSubLessonTextOrQuiz(rc *legacyRewriteCtx) error {
	return rc.remapRequired("sub_lesson_id", "course_sub_lessons")
}

func rewriteCourseSubLessonQuizOptions(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("sub_lesson_id", "course_sub_lessons"); err != nil {
		return err
	}
	if _, ok := rc.row["option_key"]; ok {
		rc.row["option_key"] = mustNewV7()
	}
	return nil
}

func rewriteCourseEditLeases(rc *legacyRewriteCtx) error {
	for _, pair := range [][2]string{
		{"course_id", "courses"},
		{"course_version_id", "course_versions"},
		{"holder_user_id", "users"},
	} {
		if err := rc.remapRequired(pair[0], pair[1]); err != nil {
			return err
		}
	}
	if _, ok := rc.row["resource_stable_id"]; ok {
		rc.row["resource_stable_id"] = mustNewV7()
	}
	if _, ok := rc.row["lease_token"]; ok {
		rc.row["lease_token"] = mustNewV7()
	}
	return nil
}

func rewriteCourseEnrollments(rc *legacyRewriteCtx) error {
	for _, pair := range [][2]string{
		{"course_id", "courses"},
		{"user_id", "users"},
		{"current_version_id", "course_versions"},
	} {
		if err := rc.remapRequired(pair[0], pair[1]); err != nil {
			return err
		}
	}
	return nil
}

func rewriteCourseProgressItems(rc *legacyRewriteCtx) error {
	if err := rc.remapRequired("enrollment_id", "course_enrollments"); err != nil {
		return err
	}
	if _, ok := rc.row["stable_content_id"]; ok {
		rc.row["stable_content_id"] = mustNewV7()
	}
	return nil
}

func (rc legacyRewriteCtx) remapRequired(col, refTable string) error {
	cur, ok := rc.row[col]
	if !ok || cur == nil {
		return nil
	}
	mapped := lookupMappedID(rc.idmap, refTable, fmt.Sprintf("%v", cur))
	if mapped == "" {
		return fmt.Errorf("missing id map for %s.%s=%v", rc.table, col, cur)
	}
	rc.row[col] = mapped
	return nil
}

func (rc legacyRewriteCtx) remapOptional(col, refTable string) error {
	if isOptionalCol(rc.row, col) {
		return nil
	}
	return rc.remapRequired(col, refTable)
}

func (rc legacyRewriteCtx) remapCourseLessonLike(parentCol, parentTable string) error {
	if err := rc.remapRequired("course_version_id", "course_versions"); err != nil {
		return err
	}
	if err := rc.remapRequired(parentCol, parentTable); err != nil {
		return err
	}
	if v, ok := rc.row["stable_id"]; ok && fmt.Sprintf("%v", v) != "" {
		rc.row["stable_id"] = mustNewV7()
	}
	return nil
}

func lookupMappedID(idmap *legacyIDMap, table, old string) string {
	if idmap == nil || idmap.Tables == nil {
		return ""
	}
	m := idmap.Tables[table]
	if m == nil {
		return ""
	}
	return m[old]
}

func isOptionalCol(row map[string]any, col string) bool {
	v, ok := row[col]
	return !ok || v == nil || strings.TrimSpace(fmt.Sprintf("%v", v)) == ""
}

func mustNewV7() string {
	id, err := uuidx.NewV7()
	if err != nil {
		return uuidx.NewV4()
	}
	return id
}
