package migrations

import (
	"strings"
	"testing"
)

func TestAuthorizationRoleBindingWildcardConstraintAddedAdditively(t *testing.T) {
	contents, err := Files.ReadFile("000035_authorization_role_binding_wildcard.up.sql")
	if err != nil {
		t.Fatalf("read wildcard constraint migration: %v", err)
	}
	migration := string(contents)

	if !strings.Contains(migration, "ALTER TABLE authorization_role_bindings") {
		t.Fatal("migration must alter authorization_role_bindings, not create a new table")
	}
	if !strings.Contains(migration, "chk_authorization_role_bindings_resource_id") {
		t.Fatal("migration must add the resource_id sentinel/UUID CHECK constraint")
	}
	if !strings.Contains(migration, "resource_id = '*'") {
		t.Fatal("constraint must allow the reserved wildcard sentinel value")
	}
	for _, table := range []string{"authorization_actions", "authorization_grants", "authorization_role_actions"} {
		if strings.Contains(migration, "ALTER TABLE "+table) {
			t.Fatalf("wildcard constraint migration must not touch unrelated table %s", table)
		}
	}
}

func TestAuthorizationRoleBindingWildcardConstraintDownDropsOnly(t *testing.T) {
	assertDownMigrationIsDataOrConstraintOnly(t,
		"000035_authorization_role_binding_wildcard.down.sql",
		"DROP CONSTRAINT IF EXISTS chk_authorization_role_bindings_resource_id",
	)
}

// assertDownMigrationIsDataOrConstraintOnly is shared by every down migration in this package
// whose only job is to undo one non-schema-shape change (a data backfill or a constraint), so
// it must contain mustContain and must never drop a table.
func assertDownMigrationIsDataOrConstraintOnly(t *testing.T, filename, mustContain string) {
	t.Helper()
	contents, err := Files.ReadFile(filename)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	migration := string(contents)
	if !strings.Contains(migration, mustContain) {
		t.Fatalf("%s must contain %q", filename, mustContain)
	}
	if strings.Contains(migration, "DROP TABLE") {
		t.Fatalf("%s must not drop any table", filename)
	}
}
