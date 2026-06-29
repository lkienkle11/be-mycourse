package infra

import (
	"testing"

	"mycourse-io-be/internal/instructor/domain"
)

func TestPlanBulkRosterWritesAllSuccess(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"u1": {ID: "u1", DisplayName: "User 1", Email: "u1@example.com"},
		"u2": {ID: "u2", DisplayName: "User 2", Email: "u2@example.com"},
	}
	existingSet := map[string]struct{}{"u1": {}}

	plan := planBulkRosterWrites([]string{"u1", "u2"}, usersByID, nil, existingSet)
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

	plan := planBulkRosterWrites([]string{"u1", "missing", "staff"}, usersByID, staffSet, nil)
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

	plan := planBulkRosterWrites([]string{"missing1", "staff"}, usersByID, staffSet, nil)
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

	plan := planBulkRosterWrites([]string{"missing", "staff"}, usersByID, staffSet, nil)
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

func TestPlanBulkRosterWritesIdempotentExisting(t *testing.T) {
	t.Parallel()
	usersByID := map[string]rosterUserRow{
		"u1": {ID: "u1", DisplayName: "User 1", Email: "u1@example.com"},
	}
	existingSet := map[string]struct{}{"u1": {}}

	plan := planBulkRosterWrites([]string{"u1"}, usersByID, nil, existingSet)
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
