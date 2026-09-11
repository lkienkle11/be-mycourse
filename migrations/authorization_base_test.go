package migrations

import (
	"strings"
	"testing"
)

func TestAuthorizationBaseGrantIndexesAreAdditive(t *testing.T) {
	contents, err := Files.ReadFile("000034_authorization_base.up.sql")
	if err != nil {
		t.Fatalf("read authorization migration: %v", err)
	}
	migration := string(contents)

	if strings.Contains(migration, "uix_authorization_grants_active") {
		t.Fatal("authorization grants must not have tuple-level active uniqueness")
	}
	for _, index := range []string{
		"CREATE INDEX idx_authorization_grants_decision",
		"CREATE INDEX idx_authorization_grants_resource",
	} {
		if !strings.Contains(migration, index) {
			t.Fatalf("missing non-unique authorization grant index %q", index)
		}
	}
}

func TestAuthorizationBaseHasNoRegisteredProviderSeed(t *testing.T) {
	contents, err := Files.ReadFile("000034_authorization_base.up.sql")
	if err != nil {
		t.Fatalf("read authorization migration: %v", err)
	}
	migration := string(contents)

	for _, table := range []string{
		"authorization_actions",
		"authorization_grants",
		"authorization_role_actions",
		"authorization_role_bindings",
	} {
		if !strings.Contains(migration, "CREATE TABLE "+table) {
			t.Fatalf("authorization base migration must create %s", table)
		}
	}
	for _, stmt := range []string{
		"INSERT INTO authorization_actions",
		"INSERT INTO authorization_grants",
		"INSERT INTO authorization_role_actions",
		"INSERT INTO authorization_role_bindings",
		"INSERT INTO course_collaborators",
	} {
		if strings.Contains(migration, stmt) {
			t.Fatalf("authorization base migration must ship as DDL only until a provider registers; found %q", stmt)
		}
	}
}
