package infra

import (
	"strings"
	"testing"

	"mycourse-io-be/internal/course/domain"
)

// TestCourseInfraSQLNeverReferencesAuthorizationRoleBindingsDirectly guards the invariant behind
// RoleBindingService.List: every course/infra SQL fragment that used to hand-write a query
// against authorization_role_bindings (collaborator listing, the instructor-candidate exclusion
// set) must instead receive an already-fetched id/binding set from RoleBindingService, never
// query that table by name itself.
func TestCourseInfraSQLNeverReferencesAuthorizationRoleBindingsDirectly(t *testing.T) {
	fragments := map[string]string{
		"collaboratorProfileSelectSQL": collaboratorProfileSelectSQL,
		"instructorCandidatesBaseSQL":  instructorCandidatesBaseSQL(),
	}
	for name, sql := range fragments {
		if strings.Contains(sql, "authorization_role_bindings") {
			t.Fatalf("%s references authorization_role_bindings directly; reads of that table must go through RoleBindingService.List", name)
		}
	}
}

func TestSortCollaboratorSourcesOwnerFirstThenGrantOrder(t *testing.T) {
	rows := []collaboratorSourceRow{
		{UserID: "editor-2", Role: domain.CollaboratorRoleEditor, SortAt: 200},
		{UserID: "owner", Role: domain.CollaboratorRoleOwner, SortAt: 50},
		{UserID: "editor-1", Role: domain.CollaboratorRoleEditor, SortAt: 100},
	}
	sortCollaboratorSources(rows)
	want := []string{"owner", "editor-1", "editor-2"}
	for i, id := range want {
		if rows[i].UserID != id {
			t.Fatalf("sortCollaboratorSources()[%d].UserID = %q, want %q (order: %#v)", i, rows[i].UserID, id, rows)
		}
	}
}

func TestPaginateCollaborators(t *testing.T) {
	all := make([]domain.Collaborator, 5)
	for i := range all {
		all[i] = domain.Collaborator{UserID: string(rune('a' + i))}
	}
	got := paginateCollaborators(all, 2, 2)
	if len(got) != 2 || got[0].UserID != "c" || got[1].UserID != "d" {
		t.Fatalf("paginateCollaborators(offset=2, perPage=2) = %#v, want [c d]", got)
	}
	if got := paginateCollaborators(all, 10, 2); len(got) != 0 {
		t.Fatalf("paginateCollaborators past the end = %#v, want empty", got)
	}
	if got := paginateCollaborators(all, 4, 10); len(got) != 1 || got[0].UserID != "e" {
		t.Fatalf("paginateCollaborators clamping the end = %#v, want [e]", got)
	}
}
