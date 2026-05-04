package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/pkg/sqlmodel"
	"mycourse-io-be/pkg/token"
	"mycourse-io-be/repository"
)

func refreshTTLForRotation(entry sqlmodel.RefreshSessionEntry) (time.Duration, error) {
	if entry.RememberMe {
		return constants.RememberMeRefreshTTL, nil
	}
	ttl := time.Until(entry.RefreshTokenExpired)
	if ttl <= 0 {
		return 0, pkgerrors.ErrRefreshTokenExpired
	}
	return ttl, nil
}

func rotateRefreshSessionTokens(user models.User, sessionStr string, entry sqlmodel.RefreshSessionEntry, newRefreshTTL time.Duration) (TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	newUUID := uuid.New().String()
	perms, permErr := userPermissionSlice(user.ID)
	if permErr != nil {
		return TokenPairResult{}, permErr
	}
	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, constants.AccessTokenTTL)
	if err != nil {
		return TokenPairResult{}, err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, newUUID, newRefreshTTL)
	if err != nil {
		return TokenPairResult{}, err
	}
	updatedEntry := sqlmodel.RefreshSessionEntry{
		RefreshTokenUUID:    newUUID,
		RememberMe:          entry.RememberMe,
		RefreshTokenExpired: time.Now().Add(newRefreshTTL),
	}
	if saveErr := repository.SaveRefreshSession(models.DB, user.ID, sessionStr, updatedEntry); saveErr != nil {
		return TokenPairResult{}, saveErr
	}
	return TokenPairResult{
		AccessToken:  at,
		RefreshToken: rt,
		SessionStr:   sessionStr,
		RefreshTTL:   newRefreshTTL,
	}, nil
}

func refreshLoadUserAndEntry(sessionStr, refreshTokenStr string) (models.User, sqlmodel.RefreshSessionEntry, error) {
	secret := setting.AppSetting.JWTSecret
	refreshClaims, err := token.ParseRefreshIgnoreExpiry(secret, refreshTokenStr)
	if err != nil {
		return models.User{}, sqlmodel.RefreshSessionEntry{}, pkgerrors.ErrInvalidSession
	}
	var user models.User
	if dbErr := models.DB.First(&user, refreshClaims.UserID).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return models.User{}, sqlmodel.RefreshSessionEntry{}, pkgerrors.ErrUserNotFound
		}
		return models.User{}, sqlmodel.RefreshSessionEntry{}, dbErr
	}
	if user.IsDisable {
		return models.User{}, sqlmodel.RefreshSessionEntry{}, pkgerrors.ErrUserDisabled
	}
	entry, ok := user.RefreshTokenSession[sessionStr]
	if !ok || entry.RefreshTokenUUID != refreshClaims.UUID {
		return models.User{}, sqlmodel.RefreshSessionEntry{}, pkgerrors.ErrInvalidSession
	}
	if time.Now().After(entry.RefreshTokenExpired) {
		return models.User{}, sqlmodel.RefreshSessionEntry{}, pkgerrors.ErrRefreshTokenExpired
	}
	return user, entry, nil
}

// RefreshSession rotates the token pair for an existing session identified by sessionStr.
// It parses the refresh token (ignoring JWT expiry — the DB record is authoritative),
// verifies the session entry, then issues fresh access + refresh tokens while keeping
// the same session string (the client's session_id cookie value is unchanged).
//
// TTL rules on rotation:
//   - remember_me=true  → new refresh TTL is always constants.RememberMeRefreshTTL (14 days from now)
//   - remember_me=false → new refresh TTL equals the remaining lifetime of the old token
func RefreshSession(sessionStr, refreshTokenStr string) (TokenPairResult, error) {
	user, entry, err := refreshLoadUserAndEntry(sessionStr, refreshTokenStr)
	if err != nil {
		return TokenPairResult{}, err
	}
	newRefreshTTL, ttlErr := refreshTTLForRotation(entry)
	if ttlErr != nil {
		return TokenPairResult{}, ttlErr
	}
	return rotateRefreshSessionTokens(user, sessionStr, entry, newRefreshTTL)
}
