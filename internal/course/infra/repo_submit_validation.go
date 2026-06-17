package infra

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/constants"
	sharedErrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/useraccess"
	sharedutils "mycourse-io-be/internal/shared/utils"
)

func (r *GormRepository) validateDraftForReview(
	ctx context.Context,
	tx *gorm.DB,
	course *courseRow,
	version *courseVersionRow,
) error {
	if err := r.validateDraftBasicInfo(ctx, tx, version); err != nil {
		return err
	}
	if err := r.validateDraftOutline(ctx, tx, version.ID); err != nil {
		return err
	}
	return r.validateDraftCollaborators(ctx, tx, course.ID)
}

func (r *GormRepository) validateDraftBasicInfo(ctx context.Context, tx *gorm.DB, version *courseVersionRow) error {
	if !hasDraftBasicInfoRequiredFields(version) {
		return domain.ErrCourseSubmitBasicInfoIncomplete
	}

	tagIDs, err := r.loadVersionRefIDs(ctx, tx, constants.TableCourseVersionTags, "tag_id", version.ID)
	if err != nil {
		return err
	}
	skillIDs, err := r.loadVersionRefIDs(ctx, tx, constants.TableCourseVersionSkills, "skill_id", version.ID)
	if err != nil {
		return err
	}
	outcomeIDs, err := r.loadVersionRefIDs(ctx, tx, constants.TableCourseVersionOutcomes, "outcome_id", version.ID)
	if err != nil {
		return err
	}
	if len(tagIDs) < 1 || len(skillIDs) < 1 || len(outcomeIDs) != 1 {
		return domain.ErrCourseSubmitBasicInfoIncomplete
	}

	if err := r.validateVersionRefs(ctx, tx, versionRefValidationInput{
		ThumbnailFileID:    version.ThumbnailFileID,
		PreviewVideoFileID: version.PreviewVideoFileID,
		CourseLevelID:      version.CourseLevelID,
		CourseTopicID:      version.CourseTopicID,
		TagIDs:             tagIDs,
		SkillIDs:           skillIDs,
		OutcomeIDs:         outcomeIDs,
	}); err != nil {
		if stderrors.Is(err, sharedErrors.ErrNotFound) || stderrors.Is(err, sharedErrors.ErrInvalidProfileMediaFile) {
			return domain.ErrCourseSubmitBasicInfoIncomplete
		}
		return err
	}
	return nil
}

func hasDraftBasicInfoRequiredFields(version *courseVersionRow) bool {
	if sharedutils.CountNonWhitespace(version.Title) < 5 {
		return false
	}
	if sharedutils.CountNonWhitespace(version.ShortDescription) < 20 {
		return false
	}
	if sharedutils.CountDeltaNonWhitespace(version.AboutCourse) < 30 {
		return false
	}
	if version.ThumbnailFileID == nil || strings.TrimSpace(*version.ThumbnailFileID) == "" {
		return false
	}
	if version.CourseLevelID == nil || strings.TrimSpace(*version.CourseLevelID) == "" {
		return false
	}
	if version.CourseTopicID == nil || strings.TrimSpace(*version.CourseTopicID) == "" {
		return false
	}
	return true
}

func (r *GormRepository) validateDraftOutline(ctx context.Context, tx *gorm.DB, versionID string) error {
	outline, err := r.loadOutlineSequential(ctx, tx, versionID)
	if err != nil {
		return err
	}
	return validateOutlineForSubmit(outline, func(subLesson domain.SubLesson) error {
		err := r.validateSubLessonContent(ctx, tx, subLessonValidationInput{
			Kind:      subLesson.Kind,
			IsPreview: subLesson.IsPreview,
			Video:     subLesson.Video,
			Text:      subLesson.Text,
			Quiz:      subLesson.Quiz,
		})
		if err == nil {
			return nil
		}
		if stderrors.Is(err, domain.ErrCourseInvalidSubLessonKind) ||
			stderrors.Is(err, domain.ErrCoursePreviewNotAllowedForQuiz) ||
			stderrors.Is(err, domain.ErrCourseQuizSingleChoiceMultipleCorrect) {
			return domain.ErrCourseSubmitInvalidSubLesson
		}
		return domain.ErrCourseSubmitInvalidSubLesson
	})
}

type collaboratorAccessSnapshot struct {
	IsDisable   bool
	DeletedAt   *int64
	BannedUntil *int64
}

func (r *GormRepository) validateDraftCollaborators(ctx context.Context, tx *gorm.DB, courseID string) error {
	collaborators, err := r.loadCollaborators(ctx, tx, courseID)
	if err != nil {
		return err
	}
	if len(collaborators) == 0 {
		return domain.ErrCourseSubmitCollaboratorRequired
	}

	for _, collaborator := range collaborators {
		var user collaboratorAccessSnapshot
		if err := tx.WithContext(ctx).
			Table(constants.TableAppUsers).
			Select("is_disable, deleted_at, banned_until").
			Where("id = ?", collaborator.UserID).
			First(&user).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrCourseCollaboratorInactive
			}
			return err
		}

		if err := useraccess.CheckAccessible(&useraccess.Snapshot{
			DeletedAt:   user.DeletedAt,
			IsDisabled:  user.IsDisable,
			BannedUntil: user.BannedUntil,
		}, timex.NowUnix()); err != nil {
			return domain.ErrCourseCollaboratorInactive
		}
		if !r.userIsInstructor(ctx, tx, collaborator.UserID) {
			return domain.ErrCourseCollaboratorInactive
		}
	}

	return nil
}

func validateOutlineForSubmit(
	outline []domain.Section,
	validateSubLesson func(subLesson domain.SubLesson) error,
) error {
	if len(outline) == 0 {
		return domain.ErrCourseSubmitOutlineIncomplete
	}
	for _, section := range outline {
		if len(section.Lessons) == 0 {
			return domain.ErrCourseSubmitOutlineIncomplete
		}
		for _, lesson := range section.Lessons {
			if len(lesson.SubLessons) == 0 {
				return domain.ErrCourseSubmitOutlineIncomplete
			}
			for _, subLesson := range lesson.SubLessons {
				if err := validateSubLesson(subLesson); err != nil {
					return domain.ErrCourseSubmitInvalidSubLesson
				}
			}
		}
	}
	return nil
}
