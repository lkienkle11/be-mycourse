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
	ID              uint
	UserID          uint
	ReviewStatus    string
	RejectionReason string
	ProfilePayload
	CreatedAt int64
	UpdatedAt int64
}

// Profile is the admin-managed instructor profile per user.
type Profile struct {
	ID     uint
	UserID uint
	ProfilePayload
	CreatedAt int64
	UpdatedAt int64
}

// RosterMember is a user with the instructor role.
type RosterMember struct {
	UserID       uint
	FullName     string
	Email        string
	Phone        string
	AvatarURL    string
	AvatarFileID string
}

// ExpertiseTopic links an instructor user to a course topic.
type ExpertiseTopic struct {
	ID        uint
	UserID    uint
	TopicID   uint
	CreatedAt int64
	UpdatedAt int64
}

// ExpertiseSkill links an instructor user to a course skill.
type ExpertiseSkill struct {
	ID        uint
	UserID    uint
	SkillID   uint
	CreatedAt int64
	UpdatedAt int64
}

// Ticket is a support thread owned by an instructor user.
type Ticket struct {
	ID        uint
	UserID    uint
	Subject   string
	Status    string
	CreatedAt int64
	UpdatedAt int64
}

// TicketMessage is one message in a ticket thread.
type TicketMessage struct {
	ID           uint
	TicketID     uint
	AuthorUserID uint
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
	UserID   uint
	Status   string
}

// SubmitApplicationInput creates or replaces a pending application.
type SubmitApplicationInput struct {
	ActorUserID uint
	ProfilePayload
}

// RejectApplicationInput rejects a pending application.
type RejectApplicationInput struct {
	ApplicationID   uint
	RejectionReason string
}

// UpsertProfileInput creates or updates a profile.
type UpsertProfileInput struct {
	UserID uint
	ProfilePayload
}
