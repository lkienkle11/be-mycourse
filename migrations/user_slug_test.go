package migrations

import (
	"strings"
	"testing"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()
	contents, err := Files.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(contents)
}

func TestUserSlugUpAddsNullableColumnConstraintsAndHistory(t *testing.T) {
	m := readMigration(t, "000043_user_slug.up.sql")
	for _, fragment := range []string{
		"ALTER TABLE users ADD COLUMN slug VARCHAR(255);",
		"chk_users_slug_format",
		"slug IS NULL OR (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$')",
		"CREATE UNIQUE INDEX uix_users_slug_active ON users (slug) WHERE deleted_at IS NULL;",
		"CREATE TABLE user_slug_history",
		"user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE",
		"CONSTRAINT uq_user_slug_history_slug UNIQUE (slug)",
		"chk_user_slug_history_slug_format",
		"CREATE INDEX idx_user_slug_history_user_id ON user_slug_history (user_id);",
	} {
		if !strings.Contains(m, fragment) {
			t.Fatalf("000043 up must contain %q", fragment)
		}
	}
	if strings.Contains(m, "ALTER COLUMN slug SET NOT NULL") {
		t.Fatal("000043 must not make users.slug NOT NULL (that is 000044, after backfill)")
	}
}

func TestUserSlugDownRemovesEverythingItAdded(t *testing.T) {
	m := readMigration(t, "000043_user_slug.down.sql")
	for _, fragment := range []string{
		"DROP TABLE IF EXISTS user_slug_history;",
		"DROP INDEX IF EXISTS uix_users_slug_active;",
		"DROP CONSTRAINT IF EXISTS chk_users_slug_format;",
		"DROP COLUMN IF EXISTS slug;",
	} {
		if !strings.Contains(m, fragment) {
			t.Fatalf("000043 down must contain %q", fragment)
		}
	}
}

func TestUserSlugNotNullMigrationPair(t *testing.T) {
	up := readMigration(t, "000044_user_slug_not_null.up.sql")
	if !strings.Contains(up, "ALTER TABLE users ALTER COLUMN slug SET NOT NULL;") {
		t.Fatal("000044 up must set users.slug NOT NULL")
	}
	down := readMigration(t, "000044_user_slug_not_null.down.sql")
	if !strings.Contains(down, "ALTER TABLE users ALTER COLUMN slug DROP NOT NULL;") {
		t.Fatal("000044 down must drop the NOT NULL")
	}
}

// The embedded runner splits files on every literal semicolon, including those
// inside comments, so a semicolon in a comment would corrupt a statement.
func TestUserSlugMigrationsHaveNoSemicolonsInComments(t *testing.T) {
	for _, name := range []string{
		"000043_user_slug.up.sql", "000043_user_slug.down.sql",
		"000044_user_slug_not_null.up.sql", "000044_user_slug_not_null.down.sql",
	} {
		for i, line := range strings.Split(readMigration(t, name), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") && strings.Contains(trimmed, ";") {
				t.Fatalf("%s line %d: comment contains a semicolon", name, i+1)
			}
		}
	}
}
