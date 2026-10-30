package infra

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mycourse-io-be/internal/auth/domain"
)

// AddSession runs in a transaction, which a dry-run handle cannot open, so the
// statements are asserted as text: the concurrency guarantees live in them.
func TestSessionSQLShape(t *testing.T) {
	cases := []struct {
		name    string
		sql     string
		want    []string
		notWant []string
	}{
		{"rotate is a conditional jsonb_set that never re-creates the session", sqlRotateSession,
			[]string{"UPDATE users", "jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, false)", "deleted_at IS NULL", "->> 'refresh_token_uuid' = ?"},
			[]string{", true)"}},
		{"add reads the user row under a lock", sqlSelectSessionsForUpdate,
			[]string{"SELECT refresh_token_session FROM users", "deleted_at IS NULL"},
			nil},
		{"add stores the map for live users only", sqlStoreSessions,
			[]string{"UPDATE users", "refresh_token_session = ?::jsonb", "WHERE id = ? AND deleted_at IS NULL"},
			nil},
		{"remove deletes one key atomically", sqlRemoveSession,
			[]string{"UPDATE users", "refresh_token_session - ?::text", "deleted_at IS NULL", "refresh_token_session -> ?::text IS NOT NULL"},
			[]string{"jsonb_set", "::jsonb"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, w := range tc.want {
				if !strings.Contains(tc.sql, w) {
					t.Fatalf("SQL %q does not contain %q", tc.sql, w)
				}
			}
			for _, n := range tc.notWant {
				if strings.Contains(tc.sql, n) {
					t.Fatalf("SQL %q must not contain %q", tc.sql, n)
				}
			}
		})
	}
	if !strings.HasSuffix(sqlSelectSessionsForUpdate, " FOR UPDATE") {
		t.Fatalf("sqlSelectSessionsForUpdate must end with FOR UPDATE: %q", sqlSelectSessionsForUpdate)
	}
}

// A full-row user save must not write the session map or the slug, so a stale
// in-memory user can not erase a rotated session or revert a slug.
func TestSaveWithDBOmitsSessionAndSlug(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=127.0.0.1 port=1 user=x dbname=x sslmode=disable", PreferSimpleProtocol: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Skipf("cannot build a dry-run gorm handle: %v", err)
	}
	var gotSQL string
	if err := db.Callback().Update().After("gorm:update").Register("test:capture", func(tx *gorm.DB) {
		gotSQL = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatal(err)
	}

	user := &domain.User{
		ID: "u1", DisplayName: "Nguyen", Slug: "nguyen",
		RefreshTokenSession: domain.RefreshTokenSessionMap{"s": {RefreshTokenUUID: "stale"}},
	}
	if err := NewGormUserRepository(db).SaveWithDB(context.Background(), db, user); err != nil {
		t.Fatalf("SaveWithDB() err = %v", err)
	}
	if !strings.HasPrefix(gotSQL, "UPDATE ") {
		t.Fatalf("expected a captured UPDATE, got %q", gotSQL)
	}
	for _, col := range []string{`"refresh_token_session"`, `"slug"`} {
		if strings.Contains(gotSQL, col) {
			t.Fatalf("full-row save must not write %s: %s", col, gotSQL)
		}
	}
	if !strings.Contains(gotSQL, `"display_name"`) {
		t.Fatalf("other columns must still be saved: %s", gotSQL)
	}
}
