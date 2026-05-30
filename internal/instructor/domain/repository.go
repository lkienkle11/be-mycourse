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
	GetApplicationByID(ctx context.Context, id uint) (*Application, error)
	GetActiveApplicationByUserID(ctx context.Context, userID uint) (*Application, error)
	UpsertPendingApplication(ctx context.Context, userID uint, p ProfilePayload) (*Application, error)
	SetApplicationReview(ctx context.Context, id uint, status, rejectionReason string) error
	DeleteApplicationsByUserID(ctx context.Context, userID uint) error
}

type ProfileRepository interface {
	ListProfiles(ctx context.Context, f ProfileFilter) ([]Profile, int64, error)
	GetProfileByUserID(ctx context.Context, userID uint) (*Profile, error)
	UpsertProfile(ctx context.Context, in UpsertProfileInput) (*Profile, error)
	DeleteProfileByUserID(ctx context.Context, userID uint) error
}

type RosterRepository interface {
	ListRoster(ctx context.Context, f RosterFilter) ([]RosterMember, int64, error)
	UserHasInstructorRole(ctx context.Context, userID uint) (bool, error)
}

type ExpertiseRepository interface {
	ListExpertise(ctx context.Context, userID uint, isTopic bool) (any, error)
	InsertExpertise(ctx context.Context, userID, refID uint, isTopic bool) (any, error)
	DeleteTopic(ctx context.Context, id uint) error
	DeleteAllTopicsForUser(ctx context.Context, userID uint) error
	ListSkills(ctx context.Context, userID uint) ([]ExpertiseSkill, error)
	DeleteSkill(ctx context.Context, id uint) error
	DeleteAllSkillsForUser(ctx context.Context, userID uint) error
}

type TicketRepository interface {
	ListTickets(ctx context.Context, f TicketFilter) ([]Ticket, int64, error)
	GetTicketByID(ctx context.Context, id uint) (*Ticket, error)
	CreateTicket(ctx context.Context, userID uint, subject string) (*Ticket, error)
	CloseTicket(ctx context.Context, id uint) error
	DeleteTicketsByUserID(ctx context.Context, userID uint) error
	ListMessages(ctx context.Context, ticketID uint) ([]TicketMessage, error)
	AddMessage(ctx context.Context, ticketID, authorUserID uint, body string) (*TicketMessage, error)
}

type InstructorDataRepository interface {
	WipeInstructorScopedData(ctx context.Context, userID uint) error
}
