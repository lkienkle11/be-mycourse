package domain

// Review status values for instructor applications.
const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
)

const (
	TicketStatusOpen   = "open"
	TicketStatusClosed = "closed"
)

const RoleNameInstructor = "instructor"
const RoleNameSysadmin = "sysadmin"
const RoleNameAdmin = "admin"

// Certificate is one credential entry stored in profile JSONB.
type Certificate struct {
	Title         string `json:"title"`
	Issuer        string `json:"issuer"`
	IssuedYear    int    `json:"issued_year"`
	CredentialURL string `json:"credential_url,omitempty"`
}

// ProfilePayload is shared by applications and managed profiles.
type ProfilePayload struct {
	Headline          string
	Bio               string
	YearsOfExperience int
	CurrentJobTitle   string
	CurrentCompany    string
	CVFileID          string
	LinkedinURL       string
	GithubURL         string
	PortfolioLinks    []string
	Certificates      []Certificate
	IntroVideoFileID  string
}

// Application is an instructor onboarding request.
type Application struct {
	ID              string
	UserID          string
	FullName        string
	AvatarURL       string
	AvatarFileID    string
	ReviewStatus    string
	RejectionReason string
	ProfilePayload
	CreatedAt int64
	UpdatedAt int64
}

// Profile is the admin-managed instructor profile per user.
type Profile struct {
	ID           string
	UserID       string
	FullName     string
	AvatarURL    string
	AvatarFileID string
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
	ID        string
	UserID    string
	Subject   string
	Status    string
	CreatedAt int64
	UpdatedAt int64
}

// TicketMessage is one message in a ticket thread.
type TicketMessage struct {
	ID           string
	TicketID     string
	AuthorUserID string
	Body         string
	CreatedAt    int64
	UpdatedAt    int64
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
	Added  []RosterMember
	Failed []RosterBulkFailure
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
}

// RejectApplicationInput rejects a pending application.
type RejectApplicationInput struct {
	ApplicationID   string
	RejectionReason string
}

// UpsertProfileInput creates or updates a profile.
type UpsertProfileInput struct {
	UserID string
	ProfilePayload
}
