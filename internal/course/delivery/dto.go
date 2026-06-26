package delivery

import (
	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/utils"
)

type createCourseRequest struct {
	Title string `json:"title" validate:"required,nonwhitespace_min=5,max=255"`
}

type updateBasicInfoRequest struct {
	ExpectedRowVersion int64    `json:"expected_row_version" validate:"required,min=1"`
	Title              string   `json:"title" validate:"required,nonwhitespace_min=5,max=255"`
	ShortDescription   string   `json:"short_description" validate:"required,nonwhitespace_min=20,max=500"`
	AboutCourse        string   `json:"about_course" validate:"required,delta_nonwhitespace_min=30"`
	ThumbnailFileID    string   `json:"thumbnail_file_id" validate:"required,uuid"`
	PreviewVideoFileID *string  `json:"preview_video_file_id" validate:"omitempty,uuid"`
	CourseLevelID      string   `json:"course_level_id" validate:"required,uuid"`
	CourseTopicID      string   `json:"course_topic_id" validate:"required,uuid"`
	TagIDs             []string `json:"tag_ids" validate:"required,min=1,dive,uuid"`
	SkillIDs           []string `json:"skill_ids" validate:"required,min=1,dive,uuid"`
	OutcomeIDs         []string `json:"outcome_ids" validate:"required,len=1,dive,uuid"`
}

type addCollaboratorsBulkRequest struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1,dive,uuid"`
	Role    string   `json:"role" binding:"omitempty,oneof=OWNER EDITOR"`
}

type collaboratorBulkFailureResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type collaboratorBulkResponse struct {
	Added  []domain.Collaborator             `json:"added"`
	Failed []collaboratorBulkFailureResponse `json:"failed"`
}

func toCollaboratorBulkResponse(result domain.CollaboratorBulkResult) collaboratorBulkResponse {
	failed := make([]collaboratorBulkFailureResponse, len(result.Failed))
	for i, row := range result.Failed {
		failed[i] = collaboratorBulkFailureResponse{UserID: row.UserID, Message: row.Message}
	}
	return collaboratorBulkResponse{Added: result.Added, Failed: failed}
}

type sectionRequest struct {
	ExpectedRowVersion int64  `json:"expected_row_version"`
	Title              string `json:"title" validate:"required,nonwhitespace_min=5,max=255"`
	Description        string `json:"description" validate:"required,delta_nonwhitespace_min=20"`
}

type lessonRequest struct {
	SectionID          string `json:"section_id" validate:"required,uuid"`
	ExpectedRowVersion int64  `json:"expected_row_version"`
	Title              string `json:"title" validate:"required,nonwhitespace_min=5,max=255"`
	Summary            string `json:"summary" validate:"required,delta_nonwhitespace_min=20"`
}

type videoRequest struct {
	MediaFileID string `json:"media_file_id" validate:"required,uuid"`
}

type textRequest struct {
	ContentDelta string `json:"content_delta"`
}

type quizOptionRequest struct {
	OptionKey string `json:"option_key" validate:"omitempty,uuid"`
	Body      string `json:"body" validate:"required"`
	IsCorrect bool   `json:"is_correct"`
}

type quizRequest struct {
	Prompt        string              `json:"prompt" validate:"required"`
	AllowMultiple bool                `json:"allow_multiple"`
	Options       []quizOptionRequest `json:"options" validate:"required,min=1,dive"`
}

type subLessonRequest struct {
	LessonID            string        `json:"lesson_id" validate:"required,uuid"`
	ExpectedRowVersion  int64         `json:"expected_row_version"`
	Title               string        `json:"title" validate:"required,nonwhitespace_min=5,max=255"`
	Kind                string        `json:"kind" validate:"required,oneof=VIDEO QUIZ TEXT"`
	IsPreview           bool          `json:"is_preview"`
	EstimatedDurationMs *int64        `json:"estimated_duration_ms,omitempty"`
	Video               *videoRequest `json:"video"`
	Text                *textRequest  `json:"text"`
	Quiz                *quizRequest  `json:"quiz"`
}

type reorderRequest struct {
	OrderedStableIDs []string `json:"ordered_stable_ids" validate:"required,min=1,dive,required,uuid"`
}

type leaseAcquireRequest struct {
	CourseVersionID  string `json:"course_version_id" validate:"required,uuid"`
	ResourceType     string `json:"resource_type" validate:"required,oneof=OUTLINE_ROOT SECTION LESSON SUB_LESSON"`
	ResourceStableID string `json:"resource_stable_id" validate:"required,uuid"`
}

type leaseHeartbeatRequest struct {
	LeaseToken string `json:"lease_token" validate:"required,uuid"`
}

type leaseReleaseRequest = leaseHeartbeatRequest

type approveDraftRequest struct {
	ApprovalNote string `json:"approval_note" validate:"required,nonwhitespace_min=5,max=500"`
}

type rejectDraftRequest struct {
	Reason string `json:"reason" validate:"required,nonwhitespace_min=5,max=500"`
}

type listReviewHistoryQuery struct {
	utils.BaseFilter
	Status string `form:"status" validate:"omitempty,oneof=APPROVED REJECTED"`
}

type coursePaginatedSearchQuery struct {
	utils.BaseFilter
	Search string `form:"search"`
}

type saveProgressRequest struct {
	StableContentID string  `json:"stable_content_id" validate:"required,uuid"`
	ContentType     string  `json:"content_type" validate:"required"`
	Status          string  `json:"status" validate:"required,oneof=NOT_STARTED IN_PROGRESS COMPLETED"`
	Score           float64 `json:"score"`
	QuizAttempt     string  `json:"quiz_attempt"`
}
