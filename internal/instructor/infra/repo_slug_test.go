package infra

import (
	"strings"
	"testing"

	"mycourse-io-be/internal/shared/userpicker"
)

func TestPopularInstructorsQuerySelectsUserSlug(t *testing.T) {
	if !strings.Contains(popularInstructorsQuery, "COALESCE(u.slug, '') AS slug") {
		t.Fatal("popularInstructorsQuery must select the user slug")
	}
	if !strings.Contains(popularInstructorsQuery, "u.id AS user_id") {
		t.Fatal("popularInstructorsQuery must keep user_id alongside the slug")
	}
}

// identityColumnsSQL is the one projection shared by the by-id loaders and the
// application/profile list queries, so asserting it covers all four.
func TestIdentityColumnsSQLSelectsSlugOnce(t *testing.T) {
	sql := identityColumnsSQL(1700000000)
	if got := strings.Count(sql, "AS slug"); got != 1 {
		t.Fatalf("identityColumnsSQL must select slug exactly once, got %d in %q", got, sql)
	}
	if !strings.Contains(sql, "u.banned_until > 1700000000") {
		t.Fatalf("identityColumnsSQL must inline the ban window timestamp, got %q", sql)
	}
}

func TestRosterCandidatesBaseSQLReusesPickerProjectionWithSlug(t *testing.T) {
	if !strings.Contains(userpicker.UserPickerSelectSQL("users"), "AS slug") {
		t.Fatal("picker projection must select slug")
	}
	if !strings.Contains(rosterCandidatesBaseSQL(), "AS slug") {
		t.Fatal("roster candidates must expose slug through the shared picker projection")
	}
}
