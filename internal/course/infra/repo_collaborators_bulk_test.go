package infra

import (
	"testing"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/useraccess"
)

const testCollaboratorEligibilityNow = int64(1_700_000_000)

func collaboratorEligibilityFor(users ...string) map[string]useraccess.AssignmentSnapshot {
	out := make(map[string]useraccess.AssignmentSnapshot, len(users))
	snap := useraccess.AssignmentSnapshot{EmailConfirmed: true}
	for _, id := range users {
		out[id] = snap
	}
	return out
}

func TestPlanBulkCollaboratorWritesAllSuccess(t *testing.T) {
	t.Parallel()
	instructorSet := map[string]struct{}{"u1": {}, "u2": {}}
	existingByUser := map[string]collaboratorRow{"u1": {ID: "collab-1", UserID: "u1"}}

	plan := planBulkCollaboratorWrites(
		[]string{"u1", "u2"},
		instructorSet,
		collaboratorEligibilityFor("u1", "u2"),
		existingByUser,
		testCollaboratorEligibilityNow,
	)
	if len(plan.failed) != 0 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if len(plan.succeededUserIDs) != 2 {
		t.Fatalf("succeeded = %v", plan.succeededUserIDs)
	}
	if len(plan.updateIDs) != 1 || plan.updateIDs[0] != "collab-1" {
		t.Fatalf("updateIDs = %v", plan.updateIDs)
	}
	if len(plan.insertUserIDs) != 1 || plan.insertUserIDs[0] != "u2" {
		t.Fatalf("insertUserIDs = %v", plan.insertUserIDs)
	}
}

func TestPlanBulkCollaboratorWritesPartialSuccess(t *testing.T) {
	t.Parallel()
	instructorSet := map[string]struct{}{"u1": {}, "u2": {}, "u3": {}}
	existingByUser := map[string]collaboratorRow{"u1": {ID: "collab-1", UserID: "u1"}}

	plan := planBulkCollaboratorWrites(
		[]string{"u1", "bad"},
		instructorSet,
		collaboratorEligibilityFor("u1"),
		existingByUser,
		testCollaboratorEligibilityNow,
	)
	if len(plan.failed) != 1 || plan.failed[0].UserID != "bad" {
		t.Fatalf("failed = %v", plan.failed)
	}
	if len(plan.succeededUserIDs) != 1 || plan.succeededUserIDs[0] != "u1" {
		t.Fatalf("succeeded = %v", plan.succeededUserIDs)
	}
}

func TestPlanBulkCollaboratorWritesAllFailed(t *testing.T) {
	t.Parallel()
	instructorSet := map[string]struct{}{"u1": {}, "u2": {}}
	existingByUser := map[string]collaboratorRow{"u1": {ID: "collab-1", UserID: "u1"}}

	plan := planBulkCollaboratorWrites(
		[]string{"bad1", "bad2"},
		instructorSet,
		collaboratorEligibilityFor(),
		existingByUser,
		testCollaboratorEligibilityNow,
	)
	if len(plan.failed) != 2 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if len(plan.succeededUserIDs) != 0 {
		t.Fatalf("succeeded = %v", plan.succeededUserIDs)
	}
	if len(plan.updateIDs) != 0 || len(plan.insertUserIDs) != 0 {
		t.Fatalf("unexpected writes update=%v insert=%v", plan.updateIDs, plan.insertUserIDs)
	}
}

func TestPlanBulkCollaboratorWritesFailedMessage(t *testing.T) {
	t.Parallel()
	instructorSet := map[string]struct{}{"u1": {}}

	plan := planBulkCollaboratorWrites(
		[]string{"bad"},
		instructorSet,
		collaboratorEligibilityFor(),
		nil,
		testCollaboratorEligibilityNow,
	)
	if len(plan.failed) != 1 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if plan.failed[0].Message != domain.ErrCourseInstructorRequired.Error() {
		t.Fatalf("message = %q", plan.failed[0].Message)
	}
}

func TestPlanBulkCollaboratorWritesInactiveUser(t *testing.T) {
	t.Parallel()
	instructorSet := map[string]struct{}{"u1": {}}
	eligibility := map[string]useraccess.AssignmentSnapshot{
		"u1": {Snapshot: useraccess.Snapshot{IsDisabled: true}, EmailConfirmed: true},
	}

	plan := planBulkCollaboratorWrites(
		[]string{"u1"},
		instructorSet,
		eligibility,
		nil,
		testCollaboratorEligibilityNow,
	)
	if len(plan.failed) != 1 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if plan.failed[0].Message != domain.ErrCourseCollaboratorInactive.Error() {
		t.Fatalf("message = %q", plan.failed[0].Message)
	}
}
