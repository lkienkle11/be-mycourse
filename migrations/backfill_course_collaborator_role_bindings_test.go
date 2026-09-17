package migrations

import (
	"strings"
	"testing"
)

func TestBackfillCourseCollaboratorRoleBindingsExcludesOwnerRows(t *testing.T) {
	contents, err := Files.ReadFile("000036_backfill_course_collaborator_role_bindings.up.sql")
	if err != nil {
		t.Fatalf("read backfill migration: %v", err)
	}
	migration := string(contents)

	if !strings.Contains(migration, "INSERT INTO authorization_role_bindings") {
		t.Fatal("migration must insert into authorization_role_bindings")
	}
	if !strings.Contains(migration, "FROM course_collaborators") {
		t.Fatal("migration must source rows from course_collaborators")
	}
	if !strings.Contains(migration, "cc.role <> 'OWNER'") {
		t.Fatal("migration must exclude OWNER-role rows: owner access is synthesized, never a stored binding")
	}
	if strings.Contains(migration, "DROP TABLE") || strings.Contains(migration, "CREATE TABLE") {
		t.Fatal("backfill migration must be data-only, no schema change")
	}
}

func TestBackfillCourseCollaboratorRoleBindingsDownIsDataOnly(t *testing.T) {
	assertDownMigrationIsDataOrConstraintOnly(t,
		"000036_backfill_course_collaborator_role_bindings.down.sql",
		"DELETE FROM authorization_role_bindings",
	)
}
