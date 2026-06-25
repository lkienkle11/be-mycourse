package infra

import "testing"

func TestCollaboratorOrderSQL(t *testing.T) {
	if got := collaboratorOrderSQL(); got == "" {
		t.Fatal("expected non-empty order clause")
	}
}
