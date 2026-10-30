package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/token"
)

const (
	sessionTestSecret  = "session-test-secret"
	sessionTestUserID  = "u1"
	sessionTestSession = "session-key-1"
	sessionTestOldUUID = "uuid-old"
)

// sessionTestRepo is an in-memory sessionRepo whose RotateSession is a
// compare-and-swap on the stored refresh_token_uuid, like the SQL it stands for.
type sessionTestRepo struct {
	domain.RefreshSessionRepository
	sessions     domain.RefreshTokenSessionMap
	expectedArgs []string
	rotateErr    error
	beforeRotate func()
}

func (r *sessionTestRepo) AddSession(context.Context, string, string, domain.RefreshSessionEntry) error {
	return nil
}

func (r *sessionTestRepo) RotateSession(_ context.Context, _ string, sessionStr, expectedUUID string, entry domain.RefreshSessionEntry) error {
	r.expectedArgs = append(r.expectedArgs, expectedUUID)
	if r.beforeRotate != nil {
		r.beforeRotate()
	}
	if r.rotateErr != nil {
		return r.rotateErr
	}
	current, ok := r.sessions[sessionStr]
	if !ok || current.RefreshTokenUUID != expectedUUID {
		return domain.ErrInvalidSession
	}
	r.sessions[sessionStr] = entry
	return nil
}

// newSessionTestService wires an AuthService over one stored session and returns
// the refresh token that matches it. JWTSecret is a process global, so it is restored.
func newSessionTestService(t *testing.T) (*AuthService, *sessionTestRepo, string) {
	t.Helper()
	previous := setting.AppSetting.JWTSecret
	setting.AppSetting.JWTSecret = sessionTestSecret
	t.Cleanup(func() { setting.AppSetting.JWTSecret = previous })

	sessions := domain.RefreshTokenSessionMap{
		sessionTestSession: {
			RefreshTokenUUID:    sessionTestOldUUID,
			RememberMe:          true,
			RefreshTokenExpired: time.Now().Add(72 * time.Hour),
		},
	}
	repo := &sessionTestRepo{sessions: sessions}
	users := &slugTestUserRepo{user: &domain.User{ID: sessionTestUserID, RefreshTokenSession: sessions}}
	refresh, err := token.GenerateRefresh(sessionTestSecret, sessionTestUserID, sessionTestOldUUID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return &AuthService{userRepo: users, sessionRepo: repo}, repo, refresh
}

func TestRefreshSessionRotatesOnce(t *testing.T) {
	svc, repo, refresh := newSessionTestService(t)

	res, err := svc.RefreshSession(context.Background(), sessionTestSession, refresh)
	if err != nil {
		t.Fatalf("RefreshSession() err = %v", err)
	}
	claims, err := token.ParseRefresh(sessionTestSecret, res.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	stored := repo.sessions[sessionTestSession]
	if claims.UUID != stored.RefreshTokenUUID || stored.RefreshTokenUUID == sessionTestOldUUID {
		t.Fatalf("token uuid %q, stored uuid %q: want equal and different from %q", claims.UUID, stored.RefreshTokenUUID, sessionTestOldUUID)
	}
	if res.SessionStr != sessionTestSession || !res.RememberMe || !stored.RememberMe {
		t.Fatalf("session id / remember_me not preserved: %+v stored %+v", res, stored)
	}
	if _, err := svc.RefreshSession(context.Background(), sessionTestSession, refresh); !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("second refresh with the old token err = %v, want ErrInvalidSession", err)
	}
}

func TestRefreshSessionPassesVerifiedUUIDToRotate(t *testing.T) {
	svc, repo, refresh := newSessionTestService(t)

	if _, err := svc.RefreshSession(context.Background(), sessionTestSession, refresh); err != nil {
		t.Fatalf("RefreshSession() err = %v", err)
	}
	if len(repo.expectedArgs) != 1 || repo.expectedArgs[0] != sessionTestOldUUID {
		t.Fatalf("RotateSession expected uuids = %v, want [%s]", repo.expectedArgs, sessionTestOldUUID)
	}
}

// The other refresh wins between the in-memory check and the write: the loser
// must fail and must not overwrite the winner's entry.
func TestRefreshSessionLosesRaceWhenUUIDChangedMeanwhile(t *testing.T) {
	svc, repo, refresh := newSessionTestService(t)
	repo.beforeRotate = func() {
		repo.sessions[sessionTestSession] = domain.RefreshSessionEntry{RefreshTokenUUID: "uuid-winner", RefreshTokenExpired: time.Now().Add(time.Hour)}
	}

	res, err := svc.RefreshSession(context.Background(), sessionTestSession, refresh)
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("err = %v, want ErrInvalidSession", err)
	}
	if res != (domain.TokenPairResult{}) {
		t.Fatalf("losing request must not return tokens, got %+v", res)
	}
	if got := repo.sessions[sessionTestSession].RefreshTokenUUID; got != "uuid-winner" {
		t.Fatalf("stored uuid = %q, the winner's entry was overwritten", got)
	}
}

func TestRefreshSessionRotateErrors(t *testing.T) {
	dbDown := errors.New("db down")
	cases := map[string]struct{ rotateErr, want error }{
		"lost race is propagated unchanged":    {domain.ErrInvalidSession, domain.ErrInvalidSession},
		"other errors are not turned into 401": {dbDown, dbDown},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc, repo, refresh := newSessionTestService(t)
			repo.rotateErr = tc.rotateErr

			res, err := svc.RefreshSession(context.Background(), sessionTestSession, refresh)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tc.want != domain.ErrInvalidSession && errors.Is(err, domain.ErrInvalidSession) {
				t.Fatalf("err = %v must not map to ErrInvalidSession", err)
			}
			if res != (domain.TokenPairResult{}) {
				t.Fatalf("no tokens expected on error, got %+v", res)
			}
		})
	}
}
