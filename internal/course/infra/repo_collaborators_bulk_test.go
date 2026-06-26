package infra

import (
	"testing"

	"mycourse-io-be/internal/course/domain"
)

func TestPlanBulkCollaboratorWritesAllSuccess(t *testing.T) {
	t.Parallel()
	instructorSet := map[string]struct{}{"u1": {}, "u2": {}}
	existingByUser := map[string]collaboratorRow{"u1": {ID: "collab-1", UserID: "u1"}}

	plan := planBulkCollaboratorWrites([]string{"u1", "u2"}, instructorSet, existingByUser)
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

	plan := planBulkCollaboratorWrites([]string{"u1", "bad"}, instructorSet, existingByUser)
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

	plan := planBulkCollaboratorWrites([]string{"bad1", "bad2"}, instructorSet, existingByUser)
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

	plan := planBulkCollaboratorWrites([]string{"bad"}, instructorSet, nil)
	if len(plan.failed) != 1 {
		t.Fatalf("failed = %v", plan.failed)
	}
	if plan.failed[0].Message != domain.ErrCourseInstructorRequired.Error() {
		t.Fatalf("message = %q", plan.failed[0].Message)
	}
}
