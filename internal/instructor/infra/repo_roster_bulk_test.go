package infra

import (
	"testing"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/useraccess"
)

const testEligibilityNow = int64(1_700_000_000)

func eligibleSnap() useraccess.AssignmentSnapshot {
	return useraccess.AssignmentSnapshot{EmailConfirmed: true}
}

func eligibilityFor(users ...string) map[string]useraccess.AssignmentSnapshot {
	out := make(map[string]useraccess.AssignmentSnapshot, len(users))
	snap := eligibleSnap()
	for _, id := range users {
		out[id] = snap
	}
	return out
}

func TestPlanBulkRosterWritesAllSuccess(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"u1": {ID: "u1", DisplayName: "User 1", Email: "u1@example.com"},
		"u2": {ID: "u2", DisplayName: "User 2", Email: "u2@example.com"},
	}
	existingSet := map[string]struct{}{"u1": {}}

	plan := planBulkRosterWrites([]string{"u1", "u2"}, usersByID, eligibilityFor("u1", "u2"), nil, existingSet, testEligibilityNow)
	if len(plan.failed) != 0 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if len(plan.succeededUserIDs) != 2 {
		t.Fatalf("succeeded = %v", plan.succeededUserIDs)
	}
	if len(plan.insertUserIDs) != 1 || plan.insertUserIDs[0] != "u2" {
		t.Fatalf("insertUserIDs = %v", plan.insertUserIDs)
	}
}

func TestPlanBulkRosterWritesPartialSuccess(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"u1": {ID: "u1", DisplayName: "User 1", Email: "u1@example.com"},
	}
	staffSet := map[string]struct{}{"staff": {}}

	plan := planBulkRosterWrites(
		[]string{"u1", "missing", "staff"},
		usersByID,
		eligibilityFor("u1", "staff"),
		staffSet,
		nil,
		testEligibilityNow,
	)
	if len(plan.failed) != 2 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if len(plan.succeededUserIDs) != 1 || plan.succeededUserIDs[0] != "u1" {
		t.Fatalf("succeeded = %v", plan.succeededUserIDs)
	}
	if len(plan.insertUserIDs) != 1 || plan.insertUserIDs[0] != "u1" {
		t.Fatalf("insertUserIDs = %v", plan.insertUserIDs)
	}
}

func TestPlanBulkRosterWritesAllFailed(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"staff": {ID: "staff", DisplayName: "Staff", Email: "staff@example.com"},
	}
	staffSet := map[string]struct{}{"staff": {}}

	plan := planBulkRosterWrites([]string{"missing1", "staff"}, usersByID, eligibilityFor("staff"), staffSet, nil, testEligibilityNow)
	if len(plan.failed) != 2 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if len(plan.succeededUserIDs) != 0 {
		t.Fatalf("succeeded = %v", plan.succeededUserIDs)
	}
	if len(plan.insertUserIDs) != 0 {
		t.Fatalf("insertUserIDs = %v", plan.insertUserIDs)
	}
}

func TestPlanBulkRosterWritesFailedMessages(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"staff": {ID: "staff", DisplayName: "Staff", Email: "staff@example.com"},
	}
	staffSet := map[string]struct{}{"staff": {}}

	plan := planBulkRosterWrites([]string{"missing", "staff"}, usersByID, eligibilityFor("staff"), staffSet, nil, testEligibilityNow)
	if len(plan.failed) != 2 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if plan.failed[0].Message != "user not found" {
		t.Fatalf("not found message = %q", plan.failed[0].Message)
	}
	if plan.failed[1].Message != domain.ErrRosterPlatformStaffUser.Error() {
		t.Fatalf("staff message = %q", plan.failed[1].Message)
	}
}

func TestPlanBulkRosterWritesIneligibleUser(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"u1": {ID: "u1", DisplayName: "User 1", Email: "u1@example.com"},
	}
	eligibility := map[string]useraccess.AssignmentSnapshot{
		"u1": {Snapshot: useraccess.Snapshot{IsDisabled: true}, EmailConfirmed: true},
	}

	plan := planBulkRosterWrites([]string{"u1"}, usersByID, eligibility, nil, nil, testEligibilityNow)
	if len(plan.failed) != 1 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if plan.failed[0].Message != useraccess.ErrUserDisabled.Error() {
		t.Fatalf("message = %q", plan.failed[0].Message)
	}
}

func TestPlanBulkRosterWritesIdempotentExisting(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"u1": {ID: "u1", DisplayName: "User 1", Email: "u1@example.com"},
	}
	existingSet := map[string]struct{}{"u1": {}}

	plan := planBulkRosterWrites([]string{"u1"}, usersByID, eligibilityFor("u1"), nil, existingSet, testEligibilityNow)
	if len(plan.failed) != 0 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if len(plan.succeededUserIDs) != 1 {
		t.Fatalf("succeeded = %v", plan.succeededUserIDs)
	}
	if len(plan.insertUserIDs) != 0 {
		t.Fatalf("insertUserIDs = %v", plan.insertUserIDs)
	}
}
