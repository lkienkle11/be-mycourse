package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/token"
)

// RefreshSession rotates the token pair for an existing session.
func (s *AuthService) RefreshSession(ctx context.Context, sessionStr, refreshTokenStr string) (domain.TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	refreshClaims, err := token.ParseRefreshIgnoreExpiry(secret, refreshTokenStr)
	if err != nil {
		return domain.TokenPairResult{}, domain.ErrInvalidSession
	}
	user, err := s.userRepo.FindByID(ctx, refreshClaims.UserID)
	if err != nil {
		if isNotFound(err) {
			return domain.TokenPairResult{}, domain.ErrUserNotFound
		}
		return domain.TokenPairResult{}, err
	}
	if err := checkUserAccessible(user, timex.NowUnix()); err != nil {
		if errors.Is(err, domain.ErrUserDisabled) || errors.Is(err, domain.ErrUserBanned) {
			return domain.TokenPairResult{}, err
		}
		return domain.TokenPairResult{}, domain.ErrUserNotFound
	}
	entry, ok := user.RefreshTokenSession[sessionStr]
	if !ok || entry.RefreshTokenUUID != refreshClaims.UUID {
		return domain.TokenPairResult{}, domain.ErrInvalidSession
	}
	if time.Now().After(entry.RefreshTokenExpired) {
		return domain.TokenPairResult{}, domain.ErrRefreshTokenExpired
	}
	var newTTL time.Duration
	if entry.RememberMe {
		newTTL = domain.RememberMeRefreshTTL
	} else {
		newTTL = time.Until(entry.RefreshTokenExpired)
		if newTTL <= 0 {
			return domain.TokenPairResult{}, domain.ErrRefreshTokenExpired
		}
	}
	return s.rotateSession(ctx, user, sessionStr, entry, newTTL)
}

// Logout revokes the refresh session identified by sessionStr and refreshTokenStr.
// Missing session is treated as success (idempotent logout).
func (s *AuthService) Logout(ctx context.Context, sessionStr, refreshTokenStr string) error {
	secret := setting.AppSetting.JWTSecret
	refreshClaims, err := token.ParseRefreshIgnoreExpiry(secret, refreshTokenStr)
	if err != nil {
		return domain.ErrInvalidSession
	}
	sessions, err := s.sessionRepo.LoadSessions(ctx, refreshClaims.UserID)
	if err != nil {
		return err
	}
	entry, ok := sessions[sessionStr]
	if !ok {
		return nil
	}
	if entry.RefreshTokenUUID != refreshClaims.UUID {
		return domain.ErrInvalidSession
	}
	if err := s.sessionRepo.RemoveSession(ctx, refreshClaims.UserID, sessionStr); err != nil {
		return err
	}
	s.delCachedMe(ctx, refreshClaims.UserID)
	return nil
}

func (s *AuthService) rotateSession(ctx context.Context, user *domain.User, sessionStr string, entry domain.RefreshSessionEntry, newTTL time.Duration) (domain.TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	newUUID := uuid.New().String()
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, domain.AccessTokenTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, newUUID, newTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	updatedEntry := domain.RefreshSessionEntry{
		RefreshTokenUUID:    newUUID,
		RememberMe:          entry.RememberMe,
		RefreshTokenExpired: time.Now().Add(newTTL),
	}
	if err := s.sessionRepo.SaveSession(ctx, user.ID, sessionStr, updatedEntry); err != nil {
		return domain.TokenPairResult{}, err
	}
	return domain.TokenPairResult{
		AccessToken:  at,
		RefreshToken: rt,
		SessionStr:   sessionStr,
		RefreshTTL:   newTTL,
	}, nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, user *domain.User, rememberMe bool, refreshTTL time.Duration) (domain.TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	sessionStr, err := token.GenerateSessionString(secret)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	sessionUUID := uuid.New().String()
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, domain.AccessTokenTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, sessionUUID, refreshTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	entry := domain.RefreshSessionEntry{
		RefreshTokenUUID:    sessionUUID,
		RememberMe:          rememberMe,
		RefreshTokenExpired: time.Now().Add(refreshTTL),
	}
	if err := s.sessionRepo.AddSession(ctx, user.ID, sessionStr, entry); err != nil {
		return domain.TokenPairResult{}, err
	}
	return domain.TokenPairResult{AccessToken: at, RefreshToken: rt, SessionStr: sessionStr, RefreshTTL: refreshTTL}, nil
}
