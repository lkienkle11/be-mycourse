package infra

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/timex"
	sharedutils "mycourse-io-be/internal/shared/utils"
)

type subLessonValidationInput struct {
	Kind      string
	IsPreview bool
	Video     *domain.VideoContent
	Text      *domain.TextContent
	Quiz      *domain.QuizContent
}

func (r *GormRepository) updateDraftStatus(ctx context.Context, courseID string, actorUserID string, fromStatus, toStatus, reason string, setSubmitted bool) (*domain.CourseDetail, error) {
	var detail *domain.CourseDetail
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := r.requireEditorAccess(ctx, tx, courseID, actorUserID); err != nil {
			return err
		}
		course, version, err := r.requireDraftVersion(ctx, tx, courseID)
		if err != nil {
			return err
		}
		if version.Status != fromStatus {
			if fromStatus == domain.VersionStatusRejected {
				return domain.ErrCourseDraftRejectedOnly
			}
			return domain.ErrCourseInvalidReviewState
		}
		if setSubmitted {
			if err := r.validateDraftForReview(ctx, tx, course, version); err != nil {
				return err
			}
		}
		updates := map[string]any{
			"status":           toStatus,
			"updated_at":       timex.NowUnix(),
			"rejection_reason": reason,
		}
		if setSubmitted {
			now := timex.NowUnix()
			updates["submitted_by_user_id"] = actorUserID
			updates["submitted_at"] = now
		}
		if toStatus == domain.VersionStatusDraft {
			updates["rejected_by_user_id"] = nil
			updates["rejected_at"] = nil
		}
		if err := tx.Model(&courseVersionRow{}).Where("id = ?", version.ID).Updates(updates).Error; err != nil {
			return err
		}
		detail, err = r.loadCourseDetail(ctx, tx, course.ID, actorUserID, true)
		return err
	})
	return detail, err
}

func (r *GormRepository) createDraftVersion(ctx context.Context, tx *gorm.DB, course *courseRow) (string, error) {
	var maxVersion int
	if err := tx.Model(&courseVersionRow{}).Where("course_id = ?", course.ID).Select("COALESCE(MAX(version_no), 0)").Scan(&maxVersion).Error; err != nil {
		return "", err
	}
	draft := &courseVersionRow{
		CourseID:   course.ID,
		VersionNo:  maxVersion + 1,
		Status:     domain.VersionStatusDraft,
		RowVersion: 1,
	}
	if course.CurrentPublishedVersionID != nil {
		live, err := r.loadVersionRow(ctx, tx, *course.CurrentPublishedVersionID)
		if err != nil {
			return "", err
		}
		draft.BasedOnVersionID = &live.ID
		draft.Title = live.Title
		draft.ShortDescription = live.ShortDescription
		draft.AboutCourse = live.AboutCourse
		draft.ThumbnailFileID = live.ThumbnailFileID
		draft.PreviewVideoFileID = live.PreviewVideoFileID
		draft.CourseLevelID = live.CourseLevelID
		draft.CourseTopicID = live.CourseTopicID
	}
	if err := touchCreateCourseEntity(ctx, tx, &draft.CreatedAt, &draft.UpdatedAt, draft); err != nil {
		return "", err
	}
	if course.CurrentPublishedVersionID != nil {
		if err := r.cloneVersionRefs(ctx, tx, *course.CurrentPublishedVersionID, draft.ID); err != nil {
			return "", err
		}
		if err := r.cloneOutline(ctx, tx, *course.CurrentPublishedVersionID, draft.ID); err != nil {
			return "", err
		}
	}
	return draft.ID, nil
}

func (r *GormRepository) cloneVersionRefs(ctx context.Context, tx *gorm.DB, fromVersionID, toVersionID string) error {
	var tagRows []courseVersionRefRow
	if err := tx.WithContext(ctx).Where("course_version_id = ?", fromVersionID).Find(&tagRows).Error; err != nil {
		return err
	}
	for i := range tagRows {
		tagRows[i].CourseVersionID = toVersionID
	}
	if len(tagRows) > 0 {
		if err := tx.Create(&tagRows).Error; err != nil {
			return err
		}
	}
	var skillRows []courseVersionSkillRefRow
	if err := tx.WithContext(ctx).Where("course_version_id = ?", fromVersionID).Find(&skillRows).Error; err != nil {
		return err
	}
	for i := range skillRows {
		skillRows[i].CourseVersionID = toVersionID
	}
	if len(skillRows) > 0 {
		if err := tx.Create(&skillRows).Error; err != nil {
			return err
		}
	}
	var outcomeRows []courseVersionOutcomeRefRow
	if err := tx.WithContext(ctx).Where("course_version_id = ?", fromVersionID).Find(&outcomeRows).Error; err != nil {
		return err
	}
	for i := range outcomeRows {
		outcomeRows[i].CourseVersionID = toVersionID
	}
	if len(outcomeRows) > 0 {
		if err := tx.Create(&outcomeRows).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *GormRepository) cloneOutline(ctx context.Context, tx *gorm.DB, fromVersionID, toVersionID string) error {
	sections, err := r.loadSectionsByVersion(ctx, tx, fromVersionID)
	if err != nil {
		return err
	}
	sectionMap, err := r.cloneSectionRows(ctx, tx, sections, toVersionID)
	if err != nil {
		return err
	}
	lessons, err := loadActiveRows[lessonRow](ctx, tx, "course_version_id = ? AND deleted_at IS NULL", fromVersionID)
	if err != nil {
		return err
	}
	lessonMap, err := r.cloneLessonRows(ctx, tx, lessons, toVersionID, sectionMap)
	if err != nil {
		return err
	}
	subLessons, err := loadActiveRows[subLessonRow](ctx, tx, "course_version_id = ? AND deleted_at IS NULL", fromVersionID)
	if err != nil {
		return err
	}
	return r.cloneSubLessonRows(ctx, tx, subLessons, toVersionID, lessonMap)
}

func (r *GormRepository) cloneSectionRows(ctx context.Context, tx *gorm.DB, sections []sectionRow, toVersionID string) (map[string]string, error) {
	sectionMap := make(map[string]string, len(sections))
	for _, section := range sections {
		clone := section
		clone.ID = ""
		clone.CourseVersionID = toVersionID
		clone.RowVersion = 1
		if err := touchCreateCourseEntity(ctx, tx, &clone.CreatedAt, &clone.UpdatedAt, &clone); err != nil {
			return nil, err
		}
		sectionMap[section.ID] = clone.ID
	}
	return sectionMap, nil
}

func (r *GormRepository) cloneLessonRows(ctx context.Context, tx *gorm.DB, lessons []lessonRow, toVersionID string, sectionMap map[string]string) (map[string]string, error) {
	lessonMap := make(map[string]string, len(lessons))
	for _, lesson := range lessons {
		clone := lesson
		clone.ID = ""
		clone.CourseVersionID = toVersionID
		clone.SectionID = sectionMap[lesson.SectionID]
		clone.RowVersion = 1
		if err := touchCreateCourseEntity(ctx, tx, &clone.CreatedAt, &clone.UpdatedAt, &clone); err != nil {
			return nil, err
		}
		lessonMap[lesson.ID] = clone.ID
	}
	return lessonMap, nil
}

func (r *GormRepository) cloneSubLessonRows(ctx context.Context, tx *gorm.DB, subLessons []subLessonRow, toVersionID string, lessonMap map[string]string) error {
	for _, subLesson := range subLessons {
		clone := subLesson
		clone.ID = ""
		clone.CourseVersionID = toVersionID
		clone.LessonID = lessonMap[subLesson.LessonID]
		clone.RowVersion = 1
		if err := touchCreateCourseEntity(ctx, tx, &clone.CreatedAt, &clone.UpdatedAt, &clone); err != nil {
			return err
		}
		if err := r.cloneSubLessonDetail(ctx, tx, subLesson.ID, clone.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *GormRepository) cloneSubLessonDetail(ctx context.Context, tx *gorm.DB, fromID, toID string) error {
	var video subLessonVideoRow
	if err := tx.WithContext(ctx).Where("sub_lesson_id = ?", fromID).First(&video).Error; err == nil {
		video.SubLessonID = toID
		if err := tx.Create(&video).Error; err != nil {
			return err
		}
	} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var text subLessonTextRow
	if err := tx.WithContext(ctx).Where("sub_lesson_id = ?", fromID).First(&text).Error; err == nil {
		text.SubLessonID = toID
		if err := tx.Create(&text).Error; err != nil {
			return err
		}
	} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var quiz subLessonQuizRow
	if err := tx.WithContext(ctx).Where("sub_lesson_id = ?", fromID).First(&quiz).Error; err == nil {
		quiz.SubLessonID = toID
		if err := tx.Create(&quiz).Error; err != nil {
			return err
		}
		var opts []subLessonQuizOptionRow
		if err := tx.WithContext(ctx).Where("sub_lesson_id = ?", fromID).Order("order_index ASC").Find(&opts).Error; err != nil {
			return err
		}
		for i := range opts {
			opts[i].ID = ""
			opts[i].SubLessonID = toID
			if err := ensureCourseRowID(&opts[i]); err != nil {
				return err
			}
		}
		if len(opts) > 0 {
			if err := tx.Create(&opts).Error; err != nil {
				return err
			}
		}
	} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (r *GormRepository) validateVersionRefs(ctx context.Context, tx *gorm.DB, in versionRefValidationInput) error {
	if err := r.validateMediaFile(ctx, tx, in.ThumbnailFileID, courseMediaKindImage); err != nil {
		return err
	}
	if err := r.validateMediaFile(ctx, tx, in.PreviewVideoFileID, constants.FileKindVideo); err != nil {
		return err
	}
	if err := r.validateLookupID(ctx, tx, constants.TableTaxonomyCourseLevels, in.CourseLevelID); err != nil {
		return err
	}
	if err := r.validateLookupID(ctx, tx, constants.TableTaxonomyCourseTopics, in.CourseTopicID); err != nil {
		return err
	}
	if err := r.validateLookupIDs(ctx, tx, constants.TableTaxonomyTags, in.TagIDs); err != nil {
		return err
	}
	if err := r.validateLookupIDs(ctx, tx, constants.TableTaxonomyCourseSkills, in.SkillIDs); err != nil {
		return err
	}
	return r.validateLookupIDs(ctx, tx, constants.TableTaxonomyCourseOutcomes, in.OutcomeIDs)
}

func (r *GormRepository) replaceVersionRefs(ctx context.Context, tx *gorm.DB, versionID string, tagIDs, skillIDs, outcomeIDs []string) error {
	if err := tx.WithContext(ctx).Where("course_version_id = ?", versionID).Delete(&courseVersionRefRow{}).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Where("course_version_id = ?", versionID).Delete(&courseVersionSkillRefRow{}).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Where("course_version_id = ?", versionID).Delete(&courseVersionOutcomeRefRow{}).Error; err != nil {
		return err
	}
	if len(tagIDs) > 0 {
		rows := make([]courseVersionRefRow, len(tagIDs))
		for i, id := range tagIDs {
			rows[i] = courseVersionRefRow{CourseVersionID: versionID, RefID: id}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
	}
	if len(skillIDs) > 0 {
		rows := make([]courseVersionSkillRefRow, len(skillIDs))
		for i, id := range skillIDs {
			rows[i] = courseVersionSkillRefRow{CourseVersionID: versionID, RefID: id}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
	}
	if len(outcomeIDs) > 0 {
		rows := make([]courseVersionOutcomeRefRow, len(outcomeIDs))
		for i, id := range outcomeIDs {
			rows[i] = courseVersionOutcomeRefRow{CourseVersionID: versionID, RefID: id}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *GormRepository) validateMediaFile(ctx context.Context, tx *gorm.DB, fileID *string, expectedKind string) error {
	if fileID == nil {
		return nil
	}
	id := strings.TrimSpace(*fileID)
	if id == "" {
		return nil
	}
	var row mediaInfoRow
	if err := tx.WithContext(ctx).Table(constants.TableMediaFiles).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&row).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrNotFound
		}
		return err
	}
	if row.Status != constants.FileStatusReady {
		return apperrors.ErrInvalidProfileMediaFile
	}
	if expectedKind == courseMediaKindImage {
		if row.Kind == constants.FileKindVideo || !strings.HasPrefix(strings.ToLower(row.MimeType), "image/") {
			return apperrors.ErrInvalidProfileMediaFile
		}
		return nil
	}
	if row.Kind != expectedKind {
		return apperrors.ErrInvalidProfileMediaFile
	}
	return nil
}

func (r *GormRepository) validateLookupID(ctx context.Context, tx *gorm.DB, table string, id *string) error {
	if id == nil || strings.TrimSpace(*id) == "" {
		return nil
	}
	var count int64
	if err := tx.WithContext(ctx).Table(table).Where("id = ? AND deleted_at IS NULL", *id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *GormRepository) validateLookupIDs(ctx context.Context, tx *gorm.DB, table string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := tx.WithContext(ctx).Table(table).Where("id IN ? AND deleted_at IS NULL", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(sharedutils.UniqueString(ids))) {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *GormRepository) validateSubLessonPayload(ctx context.Context, tx *gorm.DB, in domain.UpsertSubLessonInput) error {
	return r.validateSubLessonContent(ctx, tx, subLessonValidationInput{
		Kind:      in.Kind,
		IsPreview: in.IsPreview,
		Video:     in.Video,
		Text:      in.Text,
		Quiz:      in.Quiz,
	})
}

func (r *GormRepository) validateSubLessonContent(ctx context.Context, tx *gorm.DB, in subLessonValidationInput) error {
	kind := strings.ToUpper(strings.TrimSpace(in.Kind))
	if kind == domain.SubLessonKindQuiz && in.IsPreview {
		return domain.ErrCoursePreviewNotAllowedForQuiz
	}
	switch kind {
	case domain.SubLessonKindVideo:
		return r.validateVideoSubLesson(ctx, tx, in.Video)
	case domain.SubLessonKindText:
		return validateTextSubLesson(in.Text)
	case domain.SubLessonKindQuiz:
		return validateQuizSubLesson(in.Quiz)
	default:
		return domain.ErrCourseInvalidSubLessonKind
	}
}

func (r *GormRepository) validateVideoSubLesson(ctx context.Context, tx *gorm.DB, video *domain.VideoContent) error {
	if video == nil {
		return domain.ErrCourseInvalidSubLessonKind
	}
	mediaID := strings.TrimSpace(video.MediaFileID)
	if mediaID == "" {
		return domain.ErrCourseInvalidSubLessonKind
	}
	return r.validateMediaFile(ctx, tx, &mediaID, constants.FileKindVideo)
}

func validateTextSubLesson(text *domain.TextContent) error {
	if text == nil {
		return domain.ErrCourseInvalidSubLessonKind
	}
	content := sharedutils.NormalizeJSON(text.ContentDelta, "[]")
	var payload any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return err
	}
	if sharedutils.CountDeltaNonWhitespace(content) < 1 {
		return domain.ErrCourseInvalidSubLessonKind
	}
	return nil
}

func validateQuizSubLesson(quiz *domain.QuizContent) error {
	if quiz == nil {
		return domain.ErrCourseInvalidSubLessonKind
	}
	if strings.TrimSpace(quiz.Prompt) == "" {
		return domain.ErrCourseInvalidSubLessonKind
	}
	if len(quiz.Options) == 0 {
		return domain.ErrCourseInvalidSubLessonKind
	}
	correctCount := 0
	for _, option := range quiz.Options {
		if strings.TrimSpace(option.Body) == "" {
			return domain.ErrCourseInvalidSubLessonKind
		}
		if option.IsCorrect {
			correctCount++
		}
	}
	if correctCount == 0 {
		return domain.ErrCourseInvalidSubLessonKind
	}
	if !quiz.AllowMultiple && correctCount > 1 {
		return domain.ErrCourseQuizSingleChoiceMultipleCorrect
	}
	return nil
}

func (r *GormRepository) upsertSubLessonDetail(ctx context.Context, tx *gorm.DB, subLessonID string, in domain.UpsertSubLessonInput) error {
	now := timex.NowUnix()
	switch strings.ToUpper(strings.TrimSpace(in.Kind)) {
	case domain.SubLessonKindVideo:
		row := &subLessonVideoRow{SubLessonID: subLessonID, MediaFileID: in.Video.MediaFileID, CreatedAt: now, UpdatedAt: now}
		return tx.WithContext(ctx).Create(row).Error
	case domain.SubLessonKindText:
		row := &subLessonTextRow{SubLessonID: subLessonID, ContentDelta: sharedutils.NormalizeJSON(in.Text.ContentDelta, "[]"), CreatedAt: now, UpdatedAt: now}
		return tx.WithContext(ctx).Create(row).Error
	case domain.SubLessonKindQuiz:
		row := &subLessonQuizRow{SubLessonID: subLessonID, Prompt: strings.TrimSpace(in.Quiz.Prompt), AllowMultiple: in.Quiz.AllowMultiple, CreatedAt: now, UpdatedAt: now}
		if err := tx.WithContext(ctx).Create(row).Error; err != nil {
			return err
		}
		options := make([]subLessonQuizOptionRow, len(in.Quiz.Options))
		for i, option := range in.Quiz.Options {
			key := strings.TrimSpace(option.OptionKey)
			if key == "" {
				key = uuid.NewString()
			}
			options[i] = subLessonQuizOptionRow{
				SubLessonID: subLessonID, OptionKey: key, Body: strings.TrimSpace(option.Body),
				IsCorrect: option.IsCorrect, OrderIndex: i, CreatedAt: now, UpdatedAt: now,
			}
			if err := ensureCourseRowID(&options[i]); err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).Create(&options).Error
	default:
		return domain.ErrCourseInvalidSubLessonKind
	}
}

func (r *GormRepository) deleteSubLessonDetails(ctx context.Context, tx *gorm.DB, subLessonID string) error {
	for _, model := range []any{&subLessonVideoRow{}, &subLessonTextRow{}, &subLessonQuizOptionRow{}, &subLessonQuizRow{}} {
		if err := tx.WithContext(ctx).Where("sub_lesson_id = ?", subLessonID).Delete(model).Error; err != nil {
			return err
		}
	}
	return nil
}
