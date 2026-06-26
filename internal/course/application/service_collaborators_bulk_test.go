package application

import (
	"testing"

	"mycourse-io-be/internal/course/domain"
)

func TestPrepareCollaboratorBulkInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		userIDs  []string
		role     string
		wantIDs  []string
		wantRole string
	}{
		{
			name:     "dedupe and trim",
			userIDs:  []string{" a ", "a", "b", ""},
			role:     "",
			wantIDs:  []string{"a", "b"},
			wantRole: domain.CollaboratorRoleEditor,
		},
		{
			name:     "normalize role",
			userIDs:  []string{"u1"},
			role:     " editor ",
			wantIDs:  []string{"u1"},
			wantRole: domain.CollaboratorRoleEditor,
		},
		{
			name:     "empty input",
			userIDs:  []string{"", "  "},
			role:     "OWNER",
			wantIDs:  nil,
			wantRole: "OWNER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotIDs, gotRole := prepareCollaboratorBulkInput(tt.userIDs, tt.role)
			if gotRole != tt.wantRole {
				t.Fatalf("role = %q, want %q", gotRole, tt.wantRole)
			}
			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("ids = %v, want %v", gotIDs, tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Fatalf("ids[%d] = %q, want %q", i, gotIDs[i], tt.wantIDs[i])
				}
			}
		})
	}
}

func TestAddCollaboratorsBulkEmptyInput(t *testing.T) {
	t.Parallel()

	svc := NewCourseService(nil)
	result, err := svc.AddCollaboratorsBulk(t.Context(), "course-1", "owner-1", []string{"", " "}, "EDITOR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Added) != 0 || len(result.Failed) != 0 {
		t.Fatalf("expected empty result, got %+v", result)
	}
}
