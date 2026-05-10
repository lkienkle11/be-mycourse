package auth

import (
	"context"
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/mapping"
	authcache "mycourse-io-be/services/cache"
)

// loadUserForLogin loads the user by email, using a short-lived Redis email→id mapping when valid to skip the unique-email query.
func loadUserForLogin(ctx context.Context, email, normEmail string) (models.User, error) {
	if uid, ok := authcache.GetCachedLoginUserID(ctx, normEmail); ok {
		var u models.User
		if err := models.DB.Preload("AvatarFile").First(&u, uid).Error; err == nil && authcache.NormalizeLoginEmail(u.Email) == normEmail {
			return u, nil
		}
	}
	var user models.User
	if err := models.DB.Preload("AvatarFile").Where("email = ?", email).First(&user).Error; err != nil {
		return models.User{}, err
	}
	authcache.SetCachedLoginUserID(ctx, normEmail, user.ID)
	return user, nil
}

// Login validates credentials and returns a signed token pair plus a session string.
// rememberMe controls whether subsequent token rotations extend the refresh TTL (14 days)
// or preserve the remaining lifetime of the previous token.
func Login(email, password string, rememberMe bool) (entities.TokenPairResult, error) {
	ctx := context.Background()
	normEmail := authcache.NormalizeLoginEmail(email)
	if authcache.LoginInvalidCached(ctx, normEmail) {
		return entities.TokenPairResult{}, pkgerrors.ErrInvalidCredentials
	}

	user, dbErr := loadUserForLogin(ctx, email, normEmail)
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			authcache.SetLoginInvalidCache(ctx, normEmail)
			return entities.TokenPairResult{}, pkgerrors.ErrInvalidCredentials
		}
		return entities.TokenPairResult{}, dbErr
	}

	if user.IsDisable {
		return entities.TokenPairResult{}, pkgerrors.ErrUserDisabled
	}
	if !user.EmailConfirmed {
		return entities.TokenPairResult{}, pkgerrors.ErrEmailNotConfirmed
	}
	if bcrypt.CompareHashAndPassword([]byte(user.HashPassword), []byte(password)) != nil {
		authcache.SetLoginInvalidCache(ctx, normEmail)
		return entities.TokenPairResult{}, pkgerrors.ErrInvalidCredentials
	}
	return completeLoginSuccess(ctx, normEmail, user, rememberMe)
}

func completeLoginSuccess(ctx context.Context, normEmail string, user models.User, rememberMe bool) (entities.TokenPairResult, error) {
	refreshTTL := constants.RefreshTokenTTL
	if rememberMe {
		refreshTTL = constants.RememberMeRefreshTTL
	}
	result, err := issueTokenPair(user, rememberMe, refreshTTL)
	if err != nil {
		return entities.TokenPairResult{}, err
	}
	authcache.DelLoginInvalidCache(ctx, normEmail)
	if perms, perr := userPermissionSlice(user.ID); perr == nil {
		me := mapping.BuildMeProfileFromUser(user, perms)
		authcache.SetCachedUserMe(ctx, me)
	}
	return result, nil
}

// ConfirmEmail marks the user's email as confirmed and returns a token pair.
// remember_me is always false for email-confirmation sessions.
func ConfirmEmail(confirmToken string) (entities.TokenPairResult, error) {
	user, err := loadUserByConfirmationToken(confirmToken)
	if err != nil {
		return entities.TokenPairResult{}, err
	}
	if err := runConfirmEmailTx(&user); err != nil {
		return entities.TokenPairResult{}, err
	}
	user.EmailConfirmed = true
	user.ConfirmationToken = nil
	clearCachesAfterEmailConfirmed(context.Background(), user)
	return issueTokenPair(user, false, constants.RefreshTokenTTL)
}

// GetMe returns essential (non-sensitive) profile info for the given user along
// with their current permission codes (service/domain shape — map to dto in handlers).
func GetMe(userID uint) (*entities.MeProfile, error) {
	ctx := context.Background()
	if me, ok := authcache.GetCachedUserMe(ctx, userID); ok {
		return me, nil
	}

	var user models.User
	if err := models.DB.Preload("AvatarFile").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.ErrUserNotFound
		}
		return nil, err
	}

	perms, err := userPermissionSlice(user.ID)
	if err != nil {
		return nil, err
	}
	me := mapping.BuildMeProfileFromUser(user, perms)
	authcache.SetCachedUserMe(ctx, me)
	return me, nil
}

// isStrongPassword enforces:
// - at least 8 characters
// - at least one uppercase letter
// - at least one lowercase letter
// - at least one special character (non-letter, non-digit, non-space)
func isStrongPassword(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	var hasUpper, hasLower, hasSpecial bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasSpecial
}
