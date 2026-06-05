package delivery

type createCourseRequest struct {
	Slug  string `json:"slug" validate:"required,min=3,max=255"`
	Title string `json:"title" validate:"required,min=1,max=255"`
}

type updateBasicInfoRequest struct {
	ExpectedRowVersion int64   `json:"expected_row_version" validate:"required,min=1"`
	Title              *string `json:"title" validate:"omitempty,min=1,max=255"`
	ShortDescription   *string `json:"short_description" validate:"omitempty,max=500"`
	AboutCourse        *string `json:"about_course"`
	ThumbnailFileID    *string `json:"thumbnail_file_id" validate:"omitempty,uuid"`
	PreviewVideoFileID *string `json:"preview_video_file_id" validate:"omitempty,uuid"`
	CourseLevelID      *uint   `json:"course_level_id"`
	CourseTopicID      *uint   `json:"course_topic_id"`
	TagIDs             []uint  `json:"tag_ids"`
	SkillIDs           []uint  `json:"skill_ids"`
	OutcomeIDs         []uint  `json:"outcome_ids"`
}

type addCollaboratorRequest struct {
	UserID uint   `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"omitempty,oneof=OWNER EDITOR"`
}

type sectionRequest struct {
	ExpectedRowVersion int64  `json:"expected_row_version"`
	Title              string `json:"title" validate:"required,min=1,max=255"`
	Description        string `json:"description"`
}

type lessonRequest struct {
	SectionID          uint   `json:"section_id" validate:"required"`
	ExpectedRowVersion int64  `json:"expected_row_version"`
	Title              string `json:"title" validate:"required,min=1,max=255"`
	Summary            string `json:"summary"`
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
	LessonID           uint          `json:"lesson_id" validate:"required"`
	ExpectedRowVersion int64         `json:"expected_row_version"`
	Title              string        `json:"title" validate:"required,min=1,max=255"`
	Kind               string        `json:"kind" validate:"required,oneof=VIDEO QUIZ TEXT"`
	IsPreview          bool          `json:"is_preview"`
	Video              *videoRequest `json:"video"`
	Text               *textRequest  `json:"text"`
	Quiz               *quizRequest  `json:"quiz"`
}

type reorderRequest struct {
	OrderedStableIDs []string `json:"ordered_stable_ids" validate:"required,min=1,dive,required,uuid"`
}

type leaseAcquireRequest struct {
	CourseVersionID  uint   `json:"course_version_id" validate:"required"`
	ResourceType     string `json:"resource_type" validate:"required,oneof=OUTLINE_ROOT SECTION LESSON SUB_LESSON"`
	ResourceStableID string `json:"resource_stable_id" validate:"required,uuid"`
}

type leaseHeartbeatRequest struct {
	LeaseToken string `json:"lease_token" validate:"required,uuid"`
}

type leaseReleaseRequest = leaseHeartbeatRequest

type rejectDraftRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=2000"`
}

type saveProgressRequest struct {
	StableContentID string  `json:"stable_content_id" validate:"required,uuid"`
	ContentType     string  `json:"content_type" validate:"required"`
	Status          string  `json:"status" validate:"required,oneof=NOT_STARTED IN_PROGRESS COMPLETED"`
	Score           float64 `json:"score"`
	QuizAttempt     string  `json:"quiz_attempt"`
}
