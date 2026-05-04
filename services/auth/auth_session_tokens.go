package auth

import (
	"sort"
	"time"

	"github.com/google/uuid"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/pkg/sqlmodel"
	"mycourse-io-be/pkg/token"
	"mycourse-io-be/repository"
	"mycourse-io-be/services/rbac"
)

func generateAccessRefreshForUser(secret string, user models.User, perms []string, sessionUUID string, refreshTTL time.Duration) (access, refresh string, err error) {
	access, err = token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, constants.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = token.GenerateRefresh(secret, user.ID, sessionUUID, refreshTTL)
	return access, refresh, err
}

// issueTokenPair generates a new session string, access token and refresh token for the
// given user, persists the session entry in the DB, and returns a TokenPairResult.
func issueTokenPair(user models.User, rememberMe bool, refreshTTL time.Duration) (TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	sessionStr, err := token.GenerateSessionString(secret)
	if err != nil {
		return TokenPairResult{}, err
	}
	sessionUUID := uuid.New().String()
	perms, err := userPermissionSlice(user.ID)
	if err != nil {
		return TokenPairResult{}, err
	}
	at, rt, err := generateAccessRefreshForUser(secret, user, perms, sessionUUID, refreshTTL)
	if err != nil {
		return TokenPairResult{}, err
	}
	entry := sqlmodel.RefreshSessionEntry{
		RefreshTokenUUID:    sessionUUID,
		RememberMe:          rememberMe,
		RefreshTokenExpired: time.Now().Add(refreshTTL),
	}
	if err := repository.AddRefreshSession(models.DB, user.ID, sessionStr, entry); err != nil {
		return TokenPairResult{}, err
	}
	return TokenPairResult{AccessToken: at, RefreshToken: rt, SessionStr: sessionStr, RefreshTTL: refreshTTL}, nil
}

// userPermissionSlice returns a sorted slice of permission action strings for the user (via roles + direct grants).
func userPermissionSlice(userID uint) ([]string, error) {
	set, err := rbac.PermissionCodesForUser(userID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for action := range set {
		out = append(out, action)
	}
	sort.Strings(out)
	return out, nil
}
