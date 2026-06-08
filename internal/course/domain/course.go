package domain

import "context"

const (
	VersionStatusDraft    = "DRAFT"
	VersionStatusInReview = "IN_REVIEW"
	VersionStatusApproved = "APPROVED"
	VersionStatusRejected = "REJECTED"

	CollaboratorRoleOwner  = "OWNER"
	CollaboratorRoleEditor = "EDITOR"

	ResourceTypeOutlineRoot = "OUTLINE_ROOT"
	ResourceTypeSection     = "SECTION"
	ResourceTypeLesson      = "LESSON"
	ResourceTypeSubLesson   = "SUB_LESSON"

	SubLessonKindVideo = "VIDEO"
	SubLessonKindQuiz  = "QUIZ"
	SubLessonKindText  = "TEXT"

	ProgressStatusNotStarted = "NOT_STARTED"
	ProgressStatusInProgress = "IN_PROGRESS"
	ProgressStatusCompleted  = "COMPLETED"
)

type Course struct {
	ID                        uint   `json:"id"`
	OwnerUserID               uint   `json:"owner_user_id"`
	Slug                      string `json:"slug"`
	CurrentPublishedVersionID *uint  `json:"current_published_version_id,omitempty"`
	CurrentDraftVersionID     *uint  `json:"current_draft_version_id,omitempty"`
	CreatedAt                 int64  `json:"created_at"`
	UpdatedAt                 int64  `json:"updated_at"`
}

type CourseListItem struct {
	Course
	Title              string `json:"title"`
	ReviewStatus       string `json:"review_status"`
	VersionNo          int    `json:"version_no"`
	CollaboratorRole   string `json:"collaborator_role"`
	HasPublished       bool   `json:"has_published"`
	HasDraft           bool   `json:"has_draft"`
	ThumbnailFileID    string `json:"thumbnail_file_id,omitempty"`
	ThumbnailURL       string `json:"thumbnail_url,omitempty"`
	PreviewVideoFileID string `json:"preview_video_file_id,omitempty"`
}

type CourseVersion struct {
	ID                 uint    `json:"id"`
	CourseID           uint    `json:"course_id"`
	VersionNo          int     `json:"version_no"`
	Status             string  `json:"status"`
	BasedOnVersionID   *uint   `json:"based_on_version_id,omitempty"`
	Title              string  `json:"title"`
	ShortDescription   string  `json:"short_description"`
	AboutCourse        string  `json:"about_course"`
	ThumbnailFileID    *string `json:"thumbnail_file_id,omitempty"`
	ThumbnailURL       string  `json:"thumbnail_url,omitempty"`
	PreviewVideoFileID *string `json:"preview_video_file_id,omitempty"`
	PreviewVideoURL    string  `json:"preview_video_url,omitempty"`
	CourseLevelID      *uint   `json:"course_level_id,omitempty"`
	CourseTopicID      *uint   `json:"course_topic_id,omitempty"`
	TagIDs             []uint  `json:"tag_ids"`
	SkillIDs           []uint  `json:"skill_ids"`
	OutcomeIDs         []uint  `json:"outcome_ids"`
	RowVersion         int64   `json:"row_version"`
	SubmittedByUserID  *uint   `json:"submitted_by_user_id,omitempty"`
	SubmittedAt        *int64  `json:"submitted_at,omitempty"`
	ApprovedByUserID   *uint   `json:"approved_by_user_id,omitempty"`
	ApprovedAt         *int64  `json:"approved_at,omitempty"`
	RejectedByUserID   *uint   `json:"rejected_by_user_id,omitempty"`
	RejectedAt         *int64  `json:"rejected_at,omitempty"`
	RejectionReason    string  `json:"rejection_reason"`
	CreatedAt          int64   `json:"created_at"`
	UpdatedAt          int64   `json:"updated_at"`
}

type Collaborator struct {
	UserID       uint   `json:"user_id"`
	Role         string `json:"role"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	AvatarFileID string `json:"avatar_file_id,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
}

type Section struct {
	ID          uint     `json:"id"`
	StableID    string   `json:"stable_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	OrderIndex  int      `json:"order_index"`
	RowVersion  int64    `json:"row_version"`
	Lessons     []Lesson `json:"lessons"`
}

type Lesson struct {
	ID         uint        `json:"id"`
	StableID   string      `json:"stable_id"`
	Title      string      `json:"title"`
	Summary    string      `json:"summary"`
	OrderIndex int         `json:"order_index"`
	RowVersion int64       `json:"row_version"`
	SubLessons []SubLesson `json:"sub_lessons"`
}

type SubLesson struct {
	ID         uint          `json:"id"`
	StableID   string        `json:"stable_id"`
	Title      string        `json:"title"`
	Kind       string        `json:"kind"`
	IsPreview  bool          `json:"is_preview"`
	OrderIndex int           `json:"order_index"`
	RowVersion int64         `json:"row_version"`
	Video      *VideoContent `json:"video,omitempty"`
	Text       *TextContent  `json:"text,omitempty"`
	Quiz       *QuizContent  `json:"quiz,omitempty"`
}

type VideoContent struct {
	MediaFileID string `json:"media_file_id"`
	MediaURL    string `json:"media_url,omitempty"`
}

type TextContent struct {
	ContentDelta string `json:"content_delta"`
}

type QuizContent struct {
	Prompt        string       `json:"prompt"`
	AllowMultiple bool         `json:"allow_multiple"`
	Options       []QuizOption `json:"options"`
}

type QuizOption struct {
	ID         uint   `json:"id"`
	OptionKey  string `json:"option_key"`
	Body       string `json:"body"`
	IsCorrect  bool   `json:"is_correct"`
	OrderIndex int    `json:"order_index"`
}

type CourseDetail struct {
	Course           Course         `json:"course"`
	CollaboratorRole string         `json:"collaborator_role"`
	LiveVersion      *CourseVersion `json:"live_version,omitempty"`
	DraftVersion     *CourseVersion `json:"draft_version,omitempty"`
	Collaborators    []Collaborator `json:"collaborators"`
	Outline          []Section      `json:"outline"`
}

type Lease struct {
	ID               uint   `json:"id"`
	CourseID         uint   `json:"course_id"`
	CourseVersionID  uint   `json:"course_version_id"`
	ResourceType     string `json:"resource_type"`
	ResourceStableID string `json:"resource_stable_id"`
	HolderUserID     uint   `json:"holder_user_id"`
	LeaseToken       string `json:"lease_token"`
	ExpiresAt        int64  `json:"expires_at"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

type Enrollment struct {
	ID               uint  `json:"id"`
	CourseID         uint  `json:"course_id"`
	UserID           uint  `json:"user_id"`
	CurrentVersionID uint  `json:"current_version_id"`
	CreatedAt        int64 `json:"created_at"`
	UpdatedAt        int64 `json:"updated_at"`
}

type ProgressItem struct {
	ID               uint    `json:"id"`
	StableContentID  string  `json:"stable_content_id"`
	ContentType      string  `json:"content_type"`
	Status           string  `json:"status"`
	Score            float64 `json:"score"`
	QuizAttempt      string  `json:"quiz_attempt"`
	LastInteractedAt *int64  `json:"last_interacted_at,omitempty"`
}

type CourseProgress struct {
	Enrollment Enrollment     `json:"enrollment"`
	Items      []ProgressItem `json:"items"`
}

type Repository interface {
	ListEditableCourses(ctx context.Context, userID uint) ([]CourseListItem, error)
	CreateCourse(ctx context.Context, in CreateCourseInput) (*CourseDetail, error)
	GetCourseDetail(ctx context.Context, courseID, userID uint, includeDraft bool) (*CourseDetail, error)
	PrepareDraft(ctx context.Context, courseID, actorUserID uint) (*CourseDetail, error)
	UpdateBasicInfo(ctx context.Context, courseID, actorUserID uint, in UpdateBasicInfoInput) (*CourseDetail, error)
	DeleteCourse(ctx context.Context, courseID, actorUserID uint) error
	ListCollaborators(ctx context.Context, courseID, actorUserID uint) ([]Collaborator, error)
	AddCollaborator(ctx context.Context, courseID, actorUserID, userID uint, role string) ([]Collaborator, error)
	RemoveCollaborator(ctx context.Context, courseID, actorUserID, userID uint) ([]Collaborator, error)
	CreateSection(ctx context.Context, courseID, actorUserID uint, in UpsertSectionInput) (*Section, error)
	UpdateSection(ctx context.Context, courseID, actorUserID uint, in UpsertSectionInput) (*Section, error)
	DeleteSection(ctx context.Context, courseID, actorUserID uint, sectionID uint) ([]Section, error)
	ReorderSections(ctx context.Context, courseID, actorUserID uint, orderedStableIDs []string) ([]Section, error)
	CreateLesson(ctx context.Context, courseID, actorUserID uint, in UpsertLessonInput) (*Lesson, error)
	UpdateLesson(ctx context.Context, courseID, actorUserID uint, in UpsertLessonInput) (*Lesson, error)
	DeleteLesson(ctx context.Context, courseID, actorUserID uint, lessonID uint) ([]Section, error)
	ReorderLessons(ctx context.Context, courseID, actorUserID, sectionID uint, orderedStableIDs []string) ([]Lesson, error)
	CreateSubLesson(ctx context.Context, courseID, actorUserID uint, in UpsertSubLessonInput) (*SubLesson, error)
	UpdateSubLesson(ctx context.Context, courseID, actorUserID uint, in UpsertSubLessonInput) (*SubLesson, error)
	DeleteSubLesson(ctx context.Context, courseID, actorUserID uint, subLessonID uint) ([]Section, error)
	ReorderSubLessons(ctx context.Context, courseID, actorUserID, lessonID uint, orderedStableIDs []string) ([]SubLesson, error)
	AcquireLease(ctx context.Context, courseID, actorUserID uint, in AcquireLeaseInput) (*Lease, error)
	HeartbeatLease(ctx context.Context, courseID, actorUserID uint, in LeaseHeartbeatInput) (*Lease, error)
	ReleaseLease(ctx context.Context, courseID, actorUserID uint, in ReleaseLeaseInput) error
	SubmitForReview(ctx context.Context, courseID, actorUserID uint) (*CourseDetail, error)
	ReopenDraft(ctx context.Context, courseID, actorUserID uint) (*CourseDetail, error)
	ListPendingReviews(ctx context.Context) ([]CourseListItem, error)
	ApproveDraft(ctx context.Context, courseID, actorUserID uint) (*CourseDetail, error)
	RejectDraft(ctx context.Context, courseID, actorUserID uint, reason string) (*CourseDetail, error)
	ListPublishedCourses(ctx context.Context) ([]CourseListItem, error)
	GetLearningCourse(ctx context.Context, courseID, userID uint) (*CourseDetail, error)
	Enroll(ctx context.Context, courseID, userID uint) (*Enrollment, error)
	GetProgress(ctx context.Context, courseID, userID uint) (*CourseProgress, error)
	SaveProgress(ctx context.Context, courseID, userID uint, in SaveProgressInput) (*CourseProgress, error)
}

type CreateCourseInput struct {
	ActorUserID uint
	Slug        string
	Title       string
}

type UpdateBasicInfoInput struct {
	ActorUserID        uint
	ExpectedRowVersion int64
	Title              *string
	ShortDescription   *string
	AboutCourse        *string
	ThumbnailFileID    *string
	PreviewVideoFileID *string
	CourseLevelID      *uint
	CourseTopicID      *uint
	TagIDs             []uint
	SkillIDs           []uint
	OutcomeIDs         []uint
}

type UpsertSectionInput struct {
	SectionID          *uint
	ExpectedRowVersion int64
	Title              string
	Description        string
}

type UpsertLessonInput struct {
	LessonID           *uint
	SectionID          uint
	ExpectedRowVersion int64
	Title              string
	Summary            string
}

type UpsertSubLessonInput struct {
	SubLessonID        *uint
	LessonID           uint
	ExpectedRowVersion int64
	Title              string
	Kind               string
	IsPreview          bool
	Video              *VideoContent
	Text               *TextContent
	Quiz               *QuizContent
}

type AcquireLeaseInput struct {
	ResourceType     string
	ResourceStableID string
	CourseVersionID  uint
}

type LeaseHeartbeatInput struct {
	LeaseToken string
}

type ReleaseLeaseInput struct {
	LeaseToken string
}

type SaveProgressInput struct {
	StableContentID string
	ContentType     string
	Status          string
	Score           float64
	QuizAttempt     string
}
