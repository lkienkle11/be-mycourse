package auth

import (
	"context"
	"errors"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/brevo"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/mapping"
	"mycourse-io-be/pkg/setting"
	authcache "mycourse-io-be/services/cache"
)

// Register creates a new user and sends a confirmation email.
// Returns ErrEmailAlreadyExists when the email is already taken.
// Returns ErrWeakPassword when the password does not meet the strength requirements.
func registerAssertEmailAvailable(email string) error {
	var existing models.User
	err := models.DB.Where("email = ?", email).First(&existing).Error
	if err == nil {
		return pkgerrors.ErrEmailAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func registerInsertPendingUser(email, password, displayName string) (confirmToken string, err error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	uc, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	confirmToken = uuid.New().String()
	now := time.Now()
	user := models.User{
		UserCode:           uc.String(),
		Email:              email,
		HashPassword:       string(hash),
		DisplayName:        displayName,
		ConfirmationToken:  &confirmToken,
		ConfirmationSentAt: &now,
	}
	if err := models.DB.Create(&user).Error; err != nil {
		return "", err
	}
	return confirmToken, nil
}

func Register(email, password, displayName string) error {
	if !isStrongPassword(password) {
		return pkgerrors.ErrWeakPassword
	}
	if err := registerAssertEmailAvailable(email); err != nil {
		return err
	}
	confirmToken, err := registerInsertPendingUser(email, password, displayName)
	if err != nil {
		return err
	}
	confirmURL := setting.AppSetting.AppBaseURL + "/api/v1/auth/confirm?token=" + confirmToken
	return brevo.SendConfirmationEmail(email, displayName, confirmURL)
}

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
		me := mapping.BuildMeResponseFromUser(user, perms)
		authcache.SetCachedUserMe(ctx, me)
	}
	return result, nil
}

// ConfirmEmail marks the user's email as confirmed and returns a token pair.
// remember_me is always false for email-confirmation sessions.
func ConfirmEmail(confirmToken string) (entities.TokenPairResult, error) {
	var user models.User
	if dbErr := models.DB.Where("confirmation_token = ?", confirmToken).First(&user).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return entities.TokenPairResult{}, pkgerrors.ErrInvalidConfirmToken
		}
		return entities.TokenPairResult{}, dbErr
	}

	updates := map[string]interface{}{
		"email_confirmed":    true,
		"confirmation_token": nil,
	}
	if dbErr := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
		var learner models.Role
		if err := tx.Where("name = ?", constants.Role.Learner).First(&learner).Error; err != nil {
			return err
		}
		ur := models.UserRole{UserID: user.ID, RoleID: learner.ID}
		return tx.FirstOrCreate(&ur, models.UserRole{UserID: user.ID, RoleID: learner.ID}).Error
	}); dbErr != nil {
		return entities.TokenPairResult{}, dbErr
	}
	user.EmailConfirmed = true
	user.ConfirmationToken = nil

	return issueTokenPair(user, false, constants.RefreshTokenTTL)
}

// GetMe returns essential (non-sensitive) profile info for the given user along
// with their current permission codes.
func GetMe(userID uint) (*dto.MeResponse, error) {
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
	me := mapping.BuildMeResponseFromUser(user, perms)
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
