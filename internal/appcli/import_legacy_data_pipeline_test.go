package appcli

import (
	"reflect"
	"testing"
)

func TestLegacyCourseCollaboratorRoleBindingArgsSkipsOwnerRows(t *testing.T) {
	for _, role := range []string{"OWNER", "owner", "Owner"} {
		_, skip := legacyCourseCollaboratorRoleBindingArgs(map[string]any{
			"id": "new-id", "user_id": "new-user-id", "course_id": "new-course-id",
			"role": role, "created_at": "1700000000", "deleted_at": nil,
		})
		if !skip {
			t.Fatalf("role %q: skip = false, want true (owner access is synthesized, never a stored binding)", role)
		}
	}
}

func TestLegacyCourseCollaboratorRoleBindingArgsMapsNonOwnerRowToRoleBindingColumns(t *testing.T) {
	args, skip := legacyCourseCollaboratorRoleBindingArgs(map[string]any{
		"id": "new-id", "user_id": "new-user-id", "course_id": "new-course-id",
		"role": "EDITOR", "created_at": "1700000000", "deleted_at": nil,
	})
	if skip {
		t.Fatal("EDITOR row must not be skipped")
	}
	// Order must match the SQL's $1..$6 placeholders: id, principal_user_id, role_name,
	// resource_id, created_at, revoked_at.
	want := []any{"new-id", "new-user-id", "EDITOR", "new-course-id", "1700000000", nil}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestLegacyCourseCollaboratorRoleBindingArgsCarriesDeletedAtIntoRevokedAt(t *testing.T) {
	args, skip := legacyCourseCollaboratorRoleBindingArgs(map[string]any{
		"id": "new-id", "user_id": "new-user-id", "course_id": "new-course-id",
		"role": "EDITOR", "created_at": "1700000000", "deleted_at": "1700000500",
	})
	if skip {
		t.Fatal("EDITOR row must not be skipped")
	}
	revokedAt := args[len(args)-1]
	if revokedAt != "1700000500" {
		t.Fatalf("revoked_at arg = %#v, want the legacy row's deleted_at value carried through (a soft-deleted legacy collaborator becomes an already-revoked binding)", revokedAt)
	}
}
