package migrations

import (
	"strings"
	"testing"
)

func TestDropCourseCollaboratorsDropsExactlyThatTable(t *testing.T) {
	contents, err := Files.ReadFile("000037_drop_course_collaborators.up.sql")
	if err != nil {
		t.Fatalf("read drop migration: %v", err)
	}
	migration := string(contents)
	if !strings.Contains(migration, "DROP TABLE IF EXISTS course_collaborators") {
		t.Fatal("migration must drop course_collaborators")
	}
	for _, table := range []string{"authorization_role_bindings", "authorization_role_actions", "authorization_actions", "authorization_grants", "courses", "users"} {
		if strings.Contains(migration, "DROP TABLE "+table) || strings.Contains(migration, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("drop migration must not touch unrelated table %s", table)
		}
	}
}

func TestDropCourseCollaboratorsDownRestoresExactSchemaFrom000016(t *testing.T) {
	contents, err := Files.ReadFile("000037_drop_course_collaborators.down.sql")
	if err != nil {
		t.Fatalf("read drop-down migration: %v", err)
	}
	migration := string(contents)
	if !strings.Contains(migration, "CREATE TABLE course_collaborators") {
		t.Fatal("down migration must recreate course_collaborators")
	}
	for _, fragment := range []string{
		"course_id UUID NOT NULL REFERENCES courses (id) ON DELETE CASCADE",
		"user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE",
		"role VARCHAR(16) NOT NULL DEFAULT 'EDITOR'",
		"CREATE UNIQUE INDEX uix_course_collaborators_active",
		"CREATE INDEX idx_course_collaborators_user_active",
	} {
		if !strings.Contains(migration, fragment) {
			t.Fatalf("down migration missing expected fragment %q (must match 000016_course_management.up.sql's original shape)", fragment)
		}
	}
}
