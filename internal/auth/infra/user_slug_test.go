package infra

import (
	"errors"
	"strings"
	"testing"

	"mycourse-io-be/internal/auth/domain"
)

func TestUserRowSlugMapping(t *testing.T) {
	if row := toUserRow(&domain.User{ID: "u1"}); row.Slug != nil {
		t.Fatalf("empty domain slug must map to NULL, got %q", *row.Slug)
	}
	row := toUserRow(&domain.User{ID: "u1", Slug: "nguyen-van-a"})
	if row.Slug == nil || *row.Slug != "nguyen-van-a" {
		t.Fatalf("slug = %v, want nguyen-van-a", row.Slug)
	}
	if got := toUserDomain(&userRow{ID: "u1"}).Slug; got != "" {
		t.Fatalf("NULL slug must map to empty string, got %q", got)
	}
	slug := "stored-slug"
	if got := toUserDomain(&userRow{ID: "u1", Slug: &slug}).Slug; got != slug {
		t.Fatalf("slug = %q, want %q", got, slug)
	}
}

// A find -> save round trip must neither lose nor invent a slug in the row that
// the full-row save sees (the save itself also omits the column).
func TestUserRowSlugRoundTripThroughDomain(t *testing.T) {
	slug := "keep-me"
	back := toUserRow(toUserDomain(&userRow{ID: "u1", Slug: &slug}))
	if back.Slug == nil || *back.Slug != slug {
		t.Fatalf("round trip slug = %v, want %q", back.Slug, slug)
	}
	if back := toUserRow(toUserDomain(&userRow{ID: "u2"})); back.Slug != nil {
		t.Fatalf("round trip of a NULL slug must stay NULL, got %q", *back.Slug)
	}
}

func TestIsUserSlugDuplicateKeyIsScopedToSlugIndexes(t *testing.T) {
	for _, idx := range []string{"uix_users_slug_active", "uq_user_slug_history_slug"} {
		err := errors.New(`ERROR: duplicate key value violates unique constraint "` + idx + `"`)
		if !isUserSlugDuplicateKey(err) {
			t.Fatalf("expected slug duplicate detection for %q", idx)
		}
	}
	emailDup := errors.New(`ERROR: duplicate key value violates unique constraint "uix_users_email"`)
	if isUserSlugDuplicateKey(emailDup) {
		t.Fatal("an email unique violation must not be treated as a slug race")
	}
	if isUserSlugDuplicateKey(nil) {
		t.Fatal("nil must not match")
	}
}

func TestPlanBackfillSlugsGivesSameDisplayNameDistinctSlugs(t *testing.T) {
	rows := []backfillUserRow{
		{ID: "1", DisplayName: "Nguyễn Văn A"},
		{ID: "2", DisplayName: "Nguyễn Văn A"},
		{ID: "3", DisplayName: "Nguyễn Văn A"},
	}
	plan, err := planBackfillSlugs(rows, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]struct{}{}
	for _, a := range plan {
		if _, dup := seen[a.Slug]; dup {
			t.Fatalf("duplicate slug %q in plan %v", a.Slug, plan)
		}
		seen[a.Slug] = struct{}{}
	}
	if plan[0].Slug != "nguyen-van-a" {
		t.Fatalf("first user must get the plain base, got %q", plan[0].Slug)
	}
	for _, a := range plan[1:] {
		if !strings.HasPrefix(a.Slug, "nguyen-van-a-") {
			t.Fatalf("later users must be suffixed, got %q", a.Slug)
		}
	}
}

func TestPlanBackfillSlugsSkipsTakenAndReserved(t *testing.T) {
	rows := []backfillUserRow{
		{ID: "1", DisplayName: "Taken Name"},
		{ID: "2", DisplayName: "Admin"},
		{ID: "3", DisplayName: ""},
		{ID: "4", DisplayName: "🔥🔥"},
	}
	plan, err := planBackfillSlugs(rows, map[string]struct{}{"taken-name": {}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plan[0].Slug, "taken-name-") {
		t.Fatalf("taken base must be suffixed, got %q", plan[0].Slug)
	}
	if !strings.HasPrefix(plan[1].Slug, "admin-") {
		t.Fatalf("reserved base must be suffixed, got %q", plan[1].Slug)
	}
	for _, a := range plan[2:] {
		if !strings.HasPrefix(a.Slug, "user-") || len(a.Slug) != len("user-")+7 {
			t.Fatalf("unusable name must fall back to user-<7 chars>, got %q", a.Slug)
		}
	}
}
