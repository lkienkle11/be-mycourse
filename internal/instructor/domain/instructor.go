package domain

// Review status values for instructor applications.
const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
	ReviewStatusReturned = "returned"
)

// Years-of-experience enum codes stored in VARCHAR columns (migration 000029).
const (
	YearsUnder1Year       = "UNDER_1_YEAR"
	YearsOneToTwoYears    = "ONE_TO_TWO_YEARS"
	YearsThreeToFiveYears = "THREE_TO_FIVE_YEARS"
	YearsSixToTenYears    = "SIX_TO_TEN_YEARS"
	YearsOverTenYears     = "OVER_TEN_YEARS"
)

// ApplicationSLADays is the review window before auto-return.
const ApplicationSLADays = 5

const (
	TicketStatusOpen   = "open"
	TicketStatusClosed = "closed"
)

const RoleNameInstructor = "instructor"
const RoleNameSysadmin = "sysadmin"
const RoleNameAdmin = "admin"

// Max rejections before State H (contact admin).
const MaxApplicationRejections = 5

// Certificate is one credential entry stored in profile JSONB.
type Certificate struct {
	Title             string `json:"title"`
	Issuer            string `json:"issuer"`
	IssuedYear        int    `json:"issued_year"`
	CredentialURL     string `json:"credential_url,omitempty"`
	CertificateFileID string `json:"certificate_file_id,omitempty"`
	// CertificateFile is hydrated on read; not stored in JSONB.
	CertificateFile *MediaFileReadModel `json:"certificate_file,omitempty"`
}

// CompanySnapshot holds normalized company fields from combobox selection.
type CompanySnapshot struct {
	CurrentCompanyID          *string
	CurrentCompanyDomain      *string
	CurrentCompanyDescription *string
	CurrentCompanyLocation    *string
}

// ProfilePayload is shared by applications and managed profiles.
type ProfilePayload struct {
	Headline          string
	Bio               string
	YearsOfExperience string
	CurrentJobTitle   string
	CurrentJobTitleID string
	CurrentCompany    string
	CompanySnapshot
	CVFileID         string
	LinkedinURL      string
	GithubURL        string
	PortfolioLinks   []string
	Certificates     []Certificate
	IntroVideoFileID string
}

// RejectionRecord is one admin rejection stored in rejection_history JSONB.
type RejectionRecord struct {
	RejectedAt          int64  `json:"rejected_at"`
	RejectedByUserID    string `json:"rejected_by_user_id"`
	ReviewerDisplayName string `json:"reviewer_display_name"`
	Reason              string `json:"reason"`
}

// ApplicationTaxonomyChip is a topic/skill joined with taxonomy name for admin DTOs.
type ApplicationTaxonomyChip struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// MediaFileReadModel is a hydrated media file for API responses.
type MediaFileReadModel struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// Application is an instructor onboarding request.
type Application struct {
	ID               string
	UserID           string
	FullName         string
	DisplayName      string
	Email            string
	Phone            string
	AvatarURL        string
	AvatarFileID     string
	IsDisabled       bool
	EmailConfirmed   bool
	ReviewStatus     string
	RejectionReason  string
	SubmittedAt      int64
	ReviewDueAt      int64
	ReturnedAt       *int64
	RejectionCount   int
	RejectionHistory []RejectionRecord
	TopicIDs         []string
	SkillIDs         []string
	Topics           []ApplicationTaxonomyChip
	Skills           []ApplicationTaxonomyChip
	CVFile           *MediaFileReadModel
	IntroVideoFile   *MediaFileReadModel
	ProfilePayload
	CreatedAt int64
	UpdatedAt int64
}

// CanResubmit reports whether the user may PUT /me from returned/rejected.
func (a Application) CanResubmit() bool {
	switch a.ReviewStatus {
	case ReviewStatusReturned:
		return true
	case ReviewStatusRejected:
		return a.RejectionCount < MaxApplicationRejections
	default:
		return false
	}
}

// Profile is the admin-managed instructor profile per user.
type Profile struct {
	ID             string
	UserID         string
	FullName       string
	Email          string
	AvatarURL      string
	AvatarFileID   string
	IsDisabled     bool
	EmailConfirmed bool
	CVFile         *MediaFileReadModel
	IntroVideoFile *MediaFileReadModel
	ProfilePayload
	CreatedAt int64
	UpdatedAt int64
}

// RosterMember is a user with the instructor role.
type RosterMember struct {
	UserID       string
	FullName     string
	Email        string
	Phone        string
	AvatarURL    string
	AvatarFileID string
}

// ExpertiseTopic links an instructor user to a course topic.
type ExpertiseTopic struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	TopicID   string `json:"topic_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ExpertiseSkill links an instructor user to a course skill.
type ExpertiseSkill struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	SkillID   string `json:"skill_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// Ticket is a support thread owned by an instructor user.
type Ticket struct {
	ID           string
	UserID       string
	DisplayName  string
	Email        string
	AvatarURL    string
	AvatarFileID string
	Subject      string
	Status       string
	CreatedAt    int64
	UpdatedAt    int64
}

// TicketMessage is one message in a ticket thread.
type TicketMessage struct {
	ID             string
	TicketID       string
	AuthorUserID   string
	AuthorFullName string
	AuthorEmail    string
	Body           string
	CreatedAt      int64
	UpdatedAt      int64
}

// ApplicationFilter lists applications.
type ApplicationFilter struct {
	Page         int
	PageSize     int
	HasProfile   *bool
	ReviewStatus string
}

// RosterFilter lists roster members.
type RosterFilter struct {
	Page     int
	PageSize int
	Search   string
}

// RosterCandidate is a user eligible to receive the instructor role (no instructor/sysadmin/admin).
type RosterCandidate struct {
	UserID       string
	DisplayName  string
	Email        string
	AvatarFileID string
	AvatarURL    string
}

// RosterCandidateFilter lists roster picker candidates.
type RosterCandidateFilter struct {
	Page     int
	PageSize int
	Search   string
}

// RosterBulkFailure is one failed bulk roster add attempt.
type RosterBulkFailure struct {
	UserID  string
	Message string
}

// RosterBulkResult aggregates bulk roster add outcomes.
type RosterBulkResult struct {
	Added           []RosterMember
	Failed          []RosterBulkFailure
	InsertedUserIDs []string `json:"-"` // users with new user_roles rows; drives /me cache invalidation
}

// ProfileFilter lists profiles.
type ProfileFilter struct {
	Page     int
	PageSize int
	Search   string
}

// TicketFilter lists tickets.
type TicketFilter struct {
	Page     int
	PageSize int
	UserID   string
	Status   string
}

// SubmitApplicationInput creates or replaces a pending application.
type SubmitApplicationInput struct {
	ActorUserID string
	ProfilePayload
	TopicIDs []string
	SkillIDs []string
}

// RejectApplicationInput rejects a pending application.
type RejectApplicationInput struct {
	ApplicationID       string
	RejectionReason     string
	ReviewerUserID      string
	ReviewerDisplayName string
}

// UpsertProfileInput creates or updates a profile.
type UpsertProfileInput struct {
	UserID string
	ProfilePayload
}
