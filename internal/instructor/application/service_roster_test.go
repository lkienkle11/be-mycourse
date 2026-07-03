package application

import (
	"context"
	"testing"

	"mycourse-io-be/internal/instructor/domain"
)

type rosterBulkTestRepo struct {
	result domain.RosterBulkResult
}

func (r rosterBulkTestRepo) AddRosterBulk(context.Context, []string, uint) (domain.RosterBulkResult, error) {
	return r.result, nil
}

func (r rosterBulkTestRepo) ListApplications(context.Context, domain.ApplicationFilter) ([]domain.Application, int64, error) {
	return nil, 0, nil
}
func (r rosterBulkTestRepo) GetApplicationByID(context.Context, string) (*domain.Application, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) GetActiveApplicationByUserID(context.Context, string) (*domain.Application, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) CreateFirstApplication(context.Context, string, domain.SubmitApplicationInput) (*domain.Application, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) ResubmitApplication(context.Context, string, domain.SubmitApplicationInput) (*domain.Application, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) MarkReturnedIfDue(context.Context, string) error { return nil }
func (r rosterBulkTestRepo) RejectApplicationWithHistory(context.Context, domain.RejectApplicationInput) error {
	return nil
}
func (r rosterBulkTestRepo) ApproveApplicationCopySnapshot(context.Context, string, string) error {
	return nil
}
func (r rosterBulkTestRepo) ListApplicationTopicIDs(context.Context, string) ([]string, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) ListApplicationSkillIDs(context.Context, string) ([]string, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) ListApplicationTopics(context.Context, string) ([]domain.ApplicationTaxonomyChip, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) ListApplicationSkills(context.Context, string) ([]domain.ApplicationTaxonomyChip, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) UpsertPendingApplication(context.Context, string, domain.ProfilePayload) (*domain.Application, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) SetApplicationReview(context.Context, string, string, string) error {
	return nil
}
func (r rosterBulkTestRepo) DeleteApplicationsByUserID(context.Context, string) error { return nil }
func (r rosterBulkTestRepo) ListProfiles(context.Context, domain.ProfileFilter) ([]domain.Profile, int64, error) {
	return nil, 0, nil
}
func (r rosterBulkTestRepo) GetProfileByUserID(context.Context, string) (*domain.Profile, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) UpsertProfile(context.Context, domain.UpsertProfileInput) (*domain.Profile, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) DeleteProfileByUserID(context.Context, string) error { return nil }
func (r rosterBulkTestRepo) ListRoster(context.Context, domain.RosterFilter) ([]domain.RosterMember, int64, error) {
	return nil, 0, nil
}
func (r rosterBulkTestRepo) ListRosterCandidates(context.Context, domain.RosterCandidateFilter) ([]domain.RosterCandidate, int64, error) {
	return nil, 0, nil
}
func (r rosterBulkTestRepo) ListExpertise(context.Context, string, bool) (any, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) InsertExpertise(context.Context, string, string, bool) (any, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) DeleteTopic(context.Context, string) error            { return nil }
func (r rosterBulkTestRepo) DeleteAllTopicsForUser(context.Context, string) error { return nil }
func (r rosterBulkTestRepo) ListSkills(context.Context, string) ([]domain.ExpertiseSkill, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) DeleteSkill(context.Context, string) error            { return nil }
func (r rosterBulkTestRepo) DeleteAllSkillsForUser(context.Context, string) error { return nil }
func (r rosterBulkTestRepo) ListTickets(context.Context, domain.TicketFilter) ([]domain.Ticket, int64, error) {
	return nil, 0, nil
}
func (r rosterBulkTestRepo) GetTicketByID(context.Context, string) (*domain.Ticket, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) CreateTicket(context.Context, string, string) (*domain.Ticket, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) CreateTicketWithFirstMessage(context.Context, string, string, string) (*domain.Ticket, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) CloseTicket(context.Context, string) error           { return nil }
func (r rosterBulkTestRepo) DeleteTicketsByUserID(context.Context, string) error { return nil }
func (r rosterBulkTestRepo) ListMessages(context.Context, string) ([]domain.TicketMessage, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) AddMessage(context.Context, string, string, string) (*domain.TicketMessage, error) {
	return nil, nil
}
func (r rosterBulkTestRepo) WipeInstructorScopedData(context.Context, string) error { return nil }

type rosterBulkTestRoleMgr struct{}

func (rosterBulkTestRoleMgr) InstructorRoleID(context.Context) (uint, error)     { return 1, nil }
func (rosterBulkTestRoleMgr) AssignInstructorRole(context.Context, string) error { return nil }
func (rosterBulkTestRoleMgr) RemoveInstructorRole(context.Context, string) error { return nil }
func (rosterBulkTestRoleMgr) UserHasInstructorRole(context.Context, string) (bool, error) {
	return false, nil
}

type rosterBulkTestPerms struct{}

func (rosterBulkTestPerms) HasPermission(context.Context, string, string) bool { return false }

type rosterBulkTestMeCache struct {
	invalidated []string
}

func (c *rosterBulkTestMeCache) InvalidateUserMeCache(_ context.Context, userID string) {
	c.invalidated = append(c.invalidated, userID)
}

type rosterBulkTestHydrator struct{}

func (rosterBulkTestHydrator) ResolveAvatarURLs(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func TestAddRosterBulkInvalidatesCacheOnlyForInsertedUserIDs(t *testing.T) {
	t.Parallel()

	meCache := &rosterBulkTestMeCache{}
	svc := NewInstructorService(
		rosterBulkTestRepo{result: domain.RosterBulkResult{
			Added: []domain.RosterMember{
				{UserID: "existing"},
				{UserID: "new"},
			},
			InsertedUserIDs: []string{"new"},
		}},
		nil,
		rosterBulkTestRoleMgr{},
		rosterBulkTestPerms{},
		meCache,
		nil,
		rosterBulkTestHydrator{},
		nil,
	)

	_, err := svc.AddRosterBulk(t.Context(), []string{"existing", "new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(meCache.invalidated) != 1 || meCache.invalidated[0] != "new" {
		t.Fatalf("invalidated = %v, want [new]", meCache.invalidated)
	}
}

func TestAddRosterBulkNoCacheInvalidationWhenIdempotent(t *testing.T) {
	t.Parallel()

	meCache := &rosterBulkTestMeCache{}
	svc := NewInstructorService(
		rosterBulkTestRepo{result: domain.RosterBulkResult{
			Added:           []domain.RosterMember{{UserID: "existing"}},
			InsertedUserIDs: nil,
		}},
		nil,
		rosterBulkTestRoleMgr{},
		rosterBulkTestPerms{},
		meCache,
		nil,
		rosterBulkTestHydrator{},
		nil,
	)

	_, err := svc.AddRosterBulk(t.Context(), []string{"existing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(meCache.invalidated) != 0 {
		t.Fatalf("invalidated = %v, want none", meCache.invalidated)
	}
}
