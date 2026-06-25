package domain

import "context"

// Repository is the persistence port for the instructor bounded context.
type Repository interface {
	ApplicationRepository
	ProfileRepository
	RosterRepository
	ExpertiseRepository
	TicketRepository
	InstructorDataRepository
}

type ApplicationRepository interface {
	ListApplications(ctx context.Context, f ApplicationFilter) ([]Application, int64, error)
	GetApplicationByID(ctx context.Context, id string) (*Application, error)
	GetActiveApplicationByUserID(ctx context.Context, userID string) (*Application, error)
	UpsertPendingApplication(ctx context.Context, userID string, p ProfilePayload) (*Application, error)
	SetApplicationReview(ctx context.Context, id string, status, rejectionReason string) error
	DeleteApplicationsByUserID(ctx context.Context, userID string) error
}

type ProfileRepository interface {
	ListProfiles(ctx context.Context, f ProfileFilter) ([]Profile, int64, error)
	GetProfileByUserID(ctx context.Context, userID string) (*Profile, error)
	UpsertProfile(ctx context.Context, in UpsertProfileInput) (*Profile, error)
	DeleteProfileByUserID(ctx context.Context, userID string) error
}

type RosterRepository interface {
	ListRoster(ctx context.Context, f RosterFilter) ([]RosterMember, int64, error)
	ListRosterCandidates(ctx context.Context, f RosterCandidateFilter) ([]RosterCandidate, int64, error)
	UserHasInstructorRole(ctx context.Context, userID string) (bool, error)
	UserHasPlatformStaffRole(ctx context.Context, userID string) (bool, error)
}

type ExpertiseRepository interface {
	ListExpertise(ctx context.Context, userID string, isTopic bool) (any, error)
	InsertExpertise(ctx context.Context, userID string, refID string, isTopic bool) (any, error)
	DeleteTopic(ctx context.Context, id string) error
	DeleteAllTopicsForUser(ctx context.Context, userID string) error
	ListSkills(ctx context.Context, userID string) ([]ExpertiseSkill, error)
	DeleteSkill(ctx context.Context, id string) error
	DeleteAllSkillsForUser(ctx context.Context, userID string) error
}

type TicketRepository interface {
	ListTickets(ctx context.Context, f TicketFilter) ([]Ticket, int64, error)
	GetTicketByID(ctx context.Context, id string) (*Ticket, error)
	CreateTicket(ctx context.Context, userID string, subject string) (*Ticket, error)
	CloseTicket(ctx context.Context, id string) error
	DeleteTicketsByUserID(ctx context.Context, userID string) error
	ListMessages(ctx context.Context, ticketID string) ([]TicketMessage, error)
	AddMessage(ctx context.Context, ticketID string, authorUserID string, body string) (*TicketMessage, error)
}

type InstructorDataRepository interface {
	WipeInstructorScopedData(ctx context.Context, userID string) error
}
