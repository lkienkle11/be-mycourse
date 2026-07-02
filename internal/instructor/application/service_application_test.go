package application

import (
	"context"
	"errors"
	"testing"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

type appTestRepo struct {
	app       *domain.Application
	createErr error
	resubmit  *domain.Application
}

func (r *appTestRepo) ListApplications(context.Context, domain.ApplicationFilter) ([]domain.Application, int64, error) {
	return nil, 0, nil
}
func (r *appTestRepo) GetApplicationByID(ctx context.Context, id string) (*domain.Application, error) {
	if r.app != nil && r.app.ID == id {
		return r.app, nil
	}
	return nil, apperrors.ErrNotFound
}
func (r *appTestRepo) GetActiveApplicationByUserID(ctx context.Context, userID string) (*domain.Application, error) {
	if r.app != nil && r.app.UserID == userID {
		return r.app, nil
	}
	return nil, apperrors.ErrNotFound
}
func (r *appTestRepo) CreateFirstApplication(ctx context.Context, userID string, in domain.SubmitApplicationInput) (*domain.Application, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	a := domain.Application{ID: "app-1", UserID: userID, ReviewStatus: domain.ReviewStatusPending, ProfilePayload: in.ProfilePayload}
	r.app = &a
	return &a, nil
}
func (r *appTestRepo) ResubmitApplication(ctx context.Context, userID string, in domain.SubmitApplicationInput) (*domain.Application, error) {
	if r.resubmit != nil {
		return r.resubmit, nil
	}
	if r.app == nil {
		return nil, apperrors.ErrNotFound
	}
	r.app.ReviewStatus = domain.ReviewStatusPending
	return r.app, nil
}
func (r *appTestRepo) MarkReturnedIfDue(context.Context, string) error { return nil }
func (r *appTestRepo) SetApplicationReview(context.Context, string, string, string) error {
	return nil
}
func (r *appTestRepo) RejectApplicationWithHistory(context.Context, domain.RejectApplicationInput) error {
	return nil
}
func (r *appTestRepo) ApproveApplicationCopySnapshot(context.Context, string, string) error {
	return nil
}
func (r *appTestRepo) ListApplicationTopicIDs(context.Context, string) ([]string, error) {
	return []string{"topic-1"}, nil
}
func (r *appTestRepo) ListApplicationSkillIDs(context.Context, string) ([]string, error) {
	return []string{"skill-1"}, nil
}
func (r *appTestRepo) ListApplicationTopics(context.Context, string) ([]domain.ApplicationTaxonomyChip, error) {
	return nil, nil
}
func (r *appTestRepo) ListApplicationSkills(context.Context, string) ([]domain.ApplicationTaxonomyChip, error) {
	return nil, nil
}
func (r *appTestRepo) DeleteApplicationsByUserID(context.Context, string) error { return nil }
func (r *appTestRepo) ListProfiles(context.Context, domain.ProfileFilter) ([]domain.Profile, int64, error) {
	return nil, 0, nil
}
func (r *appTestRepo) GetProfileByUserID(context.Context, string) (*domain.Profile, error) {
	return nil, nil
}
func (r *appTestRepo) UpsertProfile(context.Context, domain.UpsertProfileInput) (*domain.Profile, error) {
	return nil, nil
}
func (r *appTestRepo) DeleteProfileByUserID(context.Context, string) error { return nil }
func (r *appTestRepo) ListRoster(context.Context, domain.RosterFilter) ([]domain.RosterMember, int64, error) {
	return nil, 0, nil
}
func (r *appTestRepo) ListRosterCandidates(context.Context, domain.RosterCandidateFilter) ([]domain.RosterCandidate, int64, error) {
	return nil, 0, nil
}
func (r *appTestRepo) AddRosterBulk(context.Context, []string, uint) (domain.RosterBulkResult, error) {
	return domain.RosterBulkResult{}, nil
}
func (r *appTestRepo) ListExpertise(context.Context, string, bool) (any, error) { return nil, nil }
func (r *appTestRepo) InsertExpertise(context.Context, string, string, bool) (any, error) {
	return nil, nil
}
func (r *appTestRepo) DeleteTopic(context.Context, string) error            { return nil }
func (r *appTestRepo) DeleteAllTopicsForUser(context.Context, string) error { return nil }
func (r *appTestRepo) ListSkills(context.Context, string) ([]domain.ExpertiseSkill, error) {
	return nil, nil
}
func (r *appTestRepo) DeleteSkill(context.Context, string) error            { return nil }
func (r *appTestRepo) DeleteAllSkillsForUser(context.Context, string) error { return nil }
func (r *appTestRepo) ListTickets(context.Context, domain.TicketFilter) ([]domain.Ticket, int64, error) {
	return nil, 0, nil
}
func (r *appTestRepo) GetTicketByID(context.Context, string) (*domain.Ticket, error) { return nil, nil }
func (r *appTestRepo) CreateTicket(context.Context, string, string) (*domain.Ticket, error) {
	return &domain.Ticket{ID: "ticket-1", Status: domain.TicketStatusOpen}, nil
}
func (r *appTestRepo) CloseTicket(context.Context, string) error           { return nil }
func (r *appTestRepo) DeleteTicketsByUserID(context.Context, string) error { return nil }
func (r *appTestRepo) ListMessages(context.Context, string) ([]domain.TicketMessage, error) {
	return nil, nil
}
func (r *appTestRepo) AddMessage(context.Context, string, string, string) (*domain.TicketMessage, error) {
	return &domain.TicketMessage{}, nil
}
func (r *appTestRepo) WipeInstructorScopedData(context.Context, string) error { return nil }

type appTestPerms struct{ blocked bool }

func (p appTestPerms) HasPermission(context.Context, string, string) bool { return p.blocked }

type appTestRoles struct{ instructor bool }

func (r appTestRoles) InstructorRoleID(context.Context) (uint, error)     { return 1, nil }
func (r appTestRoles) AssignInstructorRole(context.Context, string) error { return nil }
func (r appTestRoles) RemoveInstructorRole(context.Context, string) error { return nil }
func (r appTestRoles) UserHasInstructorRole(context.Context, string) (bool, error) {
	return r.instructor, nil
}

func validSubmitInput() domain.SubmitApplicationInput {
	bio := "I have eight years of experience building production systems, mentoring engineers, and delivering technical workshops for enterprise teams across multiple domains."
	return domain.SubmitApplicationInput{
		ActorUserID: "user-1",
		TopicIDs:    []string{"00000000-0000-0000-0000-000000000010"},
		SkillIDs:    []string{"00000000-0000-0000-0000-000000000020"},
		ProfilePayload: domain.ProfilePayload{
			Headline: "Cloud instructor", Bio: bio,
			YearsOfExperience: domain.YearsThreeToFiveYears,
			CurrentJobTitle:   "Senior Engineer", CurrentJobTitleID: "custom:senior-engineer",
			CurrentCompany: "Example Co", CVFileID: "00000000-0000-0000-0000-000000000001",
		},
	}
}

func newAppTestService(repo domain.Repository, perms PermissionChecker, roles InstructorRoleManager) *InstructorService {
	return NewInstructorService(repo, nil, roles, perms, nil, nil, rosterBulkTestHydrator{}, nil)
}

func TestSubmitApplicationBlockedByP68(t *testing.T) {
	t.Parallel()
	svc := newAppTestService(&appTestRepo{}, appTestPerms{blocked: true}, appTestRoles{})
	_, err := svc.SubmitApplication(context.Background(), validSubmitInput())
	if !errors.Is(err, domain.ErrApplicationSubmitBlocked) {
		t.Fatalf("expected submit blocked, got %v", err)
	}
}

func TestSubmitApplicationFirstSuccess(t *testing.T) {
	t.Parallel()
	repo := &appTestRepo{}
	svc := newAppTestService(repo, appTestPerms{}, appTestRoles{})
	row, err := svc.SubmitApplication(context.Background(), validSubmitInput())
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if row.ReviewStatus != domain.ReviewStatusPending {
		t.Fatalf("expected pending, got %s", row.ReviewStatus)
	}
}

func TestResubmitFromReturned(t *testing.T) {
	t.Parallel()
	repo := &appTestRepo{app: &domain.Application{
		ID: "app-1", UserID: "user-1", ReviewStatus: domain.ReviewStatusReturned,
	}}
	svc := newAppTestService(repo, appTestPerms{}, appTestRoles{})
	row, err := svc.ResubmitMyApplication(context.Background(), validSubmitInput())
	if err != nil {
		t.Fatalf("resubmit: %v", err)
	}
	if row.ReviewStatus != domain.ReviewStatusPending {
		t.Fatalf("expected pending after resubmit, got %s", row.ReviewStatus)
	}
}

func TestApplicationCanResubmit(t *testing.T) {
	t.Parallel()
	returned := domain.Application{ReviewStatus: domain.ReviewStatusReturned}
	if !returned.CanResubmit() {
		t.Fatal("returned should be resubmittable")
	}
	rejected := domain.Application{ReviewStatus: domain.ReviewStatusRejected, RejectionCount: 5}
	if rejected.CanResubmit() {
		t.Fatal("rejected with count 5 should not resubmit")
	}
}
