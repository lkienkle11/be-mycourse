package application

import (
	"errors"
	"testing"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

func TestRosterBulkClientMessage(t *testing.T) {
	t.Parallel()

	msg, ok := rosterBulkClientMessage(apperrors.ErrNotFound)
	if !ok || msg != "user not found" {
		t.Fatalf("unexpected not found mapping: %q %v", msg, ok)
	}

	msg, ok = rosterBulkClientMessage(domain.ErrRosterPlatformStaffUser)
	if !ok || msg != domain.ErrRosterPlatformStaffUser.Error() {
		t.Fatalf("unexpected platform staff mapping: %q %v", msg, ok)
	}

	msg, ok = rosterBulkClientMessage(errors.New("pq: connection refused"))
	if ok {
		t.Fatalf("expected infra error to be rejected, got %q", msg)
	}
}
