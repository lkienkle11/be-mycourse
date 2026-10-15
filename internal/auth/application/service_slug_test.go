package application

import (
	"context"
	"errors"
	"testing"

	"mycourse-io-be/internal/auth/domain"
)

// slugTestUserRepo is a minimal domain.UserRepository for UpdateMe slug flows.
type slugTestUserRepo struct {
	domain.UserRepository
	user        *domain.User
	changeCalls []string
	changeErr   error
	avatarCalls int
}

func (r *slugTestUserRepo) FindByID(context.Context, string) (*domain.User, error) {
	u := *r.user
	return &u, nil
}

func (r *slugTestUserRepo) ChangeSlug(_ context.Context, _ string, newSlug string) (string, error) {
	r.changeCalls = append(r.changeCalls, newSlug)
	if r.changeErr != nil {
		return "", r.changeErr
	}
	r.user.Slug = newSlug
	return newSlug, nil
}

func (r *slugTestUserRepo) UpdateAvatar(context.Context, string, *string) error {
	r.avatarCalls++
	return nil
}

func newSlugTestService(repo *slugTestUserRepo) *AuthService {
	return &AuthService{
		userRepo:       repo,
		permReader:     roleTestPermissionReader{},
		roleNameReader: &roleTestNameReader{},
	}
}

func strPtr(s string) *string { return &s }

func TestUpdateMeSlugChangesSlugAndReturnsProfile(t *testing.T) {
	repo := &slugTestUserRepo{user: &domain.User{ID: "u1", Slug: "old-slug"}}
	svc := newSlugTestService(repo)

	me, err := svc.UpdateMe(context.Background(), "u1", domain.UpdateMeParams{Slug: strPtr("new-slug")})
	if err != nil {
		t.Fatalf("UpdateMe() err = %v", err)
	}
	if len(repo.changeCalls) != 1 || repo.changeCalls[0] != "new-slug" {
		t.Fatalf("ChangeSlug calls = %v, want [new-slug]", repo.changeCalls)
	}
	if me.Slug != "new-slug" {
		t.Fatalf("profile slug = %q, want new-slug", me.Slug)
	}
	if repo.avatarCalls != 0 {
		t.Fatalf("avatar must not be touched by a slug-only patch, got %d calls", repo.avatarCalls)
	}
}

func TestUpdateMeSlugEqualToCurrentIsNoOp(t *testing.T) {
	repo := &slugTestUserRepo{user: &domain.User{ID: "u1", Slug: "same-slug"}}
	svc := newSlugTestService(repo)

	if _, err := svc.UpdateMe(context.Background(), "u1", domain.UpdateMeParams{Slug: strPtr("same-slug")}); err != nil {
		t.Fatalf("UpdateMe() err = %v", err)
	}
	if len(repo.changeCalls) != 0 {
		t.Fatalf("ChangeSlug must not run for an unchanged slug, got %v", repo.changeCalls)
	}
}

func TestUpdateMeSlugReservedIsRejectedBeforeAnyWrite(t *testing.T) {
	repo := &slugTestUserRepo{user: &domain.User{ID: "u1", Slug: "old-slug"}}
	svc := newSlugTestService(repo)

	_, err := svc.UpdateMe(context.Background(), "u1", domain.UpdateMeParams{
		AvatarFileID: strPtr("00000000-0000-0000-0000-000000000001"),
		Slug:         strPtr("admin"),
	})
	if !errors.Is(err, domain.ErrUserSlugReserved) {
		t.Fatalf("UpdateMe() err = %v, want ErrUserSlugReserved", err)
	}
	if repo.avatarCalls != 0 || len(repo.changeCalls) != 0 {
		t.Fatalf("a rejected slug must not apply the avatar: avatar=%d change=%v", repo.avatarCalls, repo.changeCalls)
	}
}

func TestUpdateMeNothingSetReturnsProfileWithoutWrites(t *testing.T) {
	repo := &slugTestUserRepo{user: &domain.User{ID: "u1", Slug: "old-slug"}}
	svc := newSlugTestService(repo)

	me, err := svc.UpdateMe(context.Background(), "u1", domain.UpdateMeParams{})
	if err != nil {
		t.Fatalf("UpdateMe() err = %v", err)
	}
	if me.Slug != "old-slug" || repo.avatarCalls != 0 || len(repo.changeCalls) != 0 {
		t.Fatalf("unexpected writes or profile: slug=%q avatar=%d change=%v", me.Slug, repo.avatarCalls, repo.changeCalls)
	}
}

func TestUpdateMeSlugPropagatesRepositoryError(t *testing.T) {
	boom := errors.New("db down")
	repo := &slugTestUserRepo{user: &domain.User{ID: "u1", Slug: "old"}, changeErr: boom}
	svc := newSlugTestService(repo)

	if _, err := svc.UpdateMe(context.Background(), "u1", domain.UpdateMeParams{Slug: strPtr("new-slug")}); !errors.Is(err, boom) {
		t.Fatalf("UpdateMe() err = %v, want %v", err, boom)
	}
}

func TestBuildMeProfileIncludesSlug(t *testing.T) {
	me := buildMeProfile(&domain.User{ID: "u1", Slug: "nguyen-van-a"}, nil, nil)
	if me.Slug != "nguyen-van-a" {
		t.Fatalf("slug = %q, want nguyen-van-a", me.Slug)
	}
}

type rejectingMediaValidator struct{ err error }

func (v rejectingMediaValidator) ValidateProfileImageFile(string) error { return v.err }

// A rejected avatar must be caught before the slug write, so a PATCH carrying
// both fields never leaves the slug changed while reporting a failure.
func TestUpdateMeAvatarValidationFailsBeforeSlugWrite(t *testing.T) {
	bad := errors.New("not an image")
	repo := &slugTestUserRepo{user: &domain.User{ID: "u1", Slug: "old-slug"}}
	svc := newSlugTestService(repo)
	svc.mediaValidator = rejectingMediaValidator{err: bad}

	_, err := svc.UpdateMe(context.Background(), "u1", domain.UpdateMeParams{
		AvatarFileID: strPtr("00000000-0000-0000-0000-000000000001"),
		Slug:         strPtr("new-slug"),
	})
	if !errors.Is(err, bad) {
		t.Fatalf("UpdateMe() err = %v, want %v", err, bad)
	}
	if len(repo.changeCalls) != 0 || repo.avatarCalls != 0 {
		t.Fatalf("no write may happen after a rejected avatar: change=%v avatar=%d", repo.changeCalls, repo.avatarCalls)
	}
}
