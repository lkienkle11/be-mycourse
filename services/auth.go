package services

import (
	"errors"
	"sort"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/brevo"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/pkg/token"
)

// Auth token lifetimes — exported so the HTTP layer can derive cookie MaxAge from the same value.
const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

// Sentinel errors returned by auth functions.
var (
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrWeakPassword        = errors.New("password does not meet requirements")
	ErrEmailNotConfirmed   = errors.New("email not confirmed")
	ErrUserDisabled        = errors.New("user account is disabled")
	ErrInvalidConfirmToken = errors.New("invalid or expired confirmation token")
	ErrUserNotFound        = errors.New("user not found")
)

// Register creates a new user and sends a confirmation email.
// Returns ErrEmailAlreadyExists when the email is already taken.
// Returns ErrWeakPassword when the password does not meet the strength requirements.
func Register(email, password, displayName string) error {
	if !isStrongPassword(password) {
		return ErrWeakPassword
	}

	var existing models.User
	err := models.DB.Where("email = ?", email).First(&existing).Error
	if err == nil {
		return ErrEmailAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	uc, err := uuid.NewV7()
	if err != nil {
		return err
	}

	confirmToken := uuid.New().String()
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
		return err
	}

	confirmURL := setting.AppSetting.AppBaseURL + "/api/v1/auth/confirm?token=" + confirmToken
	return brevo.SendConfirmationEmail(email, displayName, confirmURL)
}

// Login validates credentials and returns a signed access + refresh token pair.
func Login(email, password string) (accessToken, refreshToken string, err error) {
	var user models.User
	if dbErr := models.DB.Where("email = ?", email).First(&user).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", dbErr
	}

	if user.IsDisable {
		return "", "", ErrUserDisabled
	}
	if !user.EmailConfirmed {
		return "", "", ErrEmailNotConfirmed
	}
	if bcrypt.CompareHashAndPassword([]byte(user.HashPassword), []byte(password)) != nil {
		return "", "", ErrInvalidCredentials
	}

	return issueTokenPair(user)
}

// ConfirmEmail marks the user's email as confirmed and returns a token pair.
func ConfirmEmail(confirmToken string) (accessToken, refreshToken string, err error) {
	var user models.User
	if dbErr := models.DB.Where("confirmation_token = ?", confirmToken).First(&user).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return "", "", ErrInvalidConfirmToken
		}
		return "", "", dbErr
	}

	updates := map[string]interface{}{
		"email_confirmed":   true,
		"confirmation_token": nil,
	}
	if dbErr := models.DB.Model(&user).Updates(updates).Error; dbErr != nil {
		return "", "", dbErr
	}
	user.EmailConfirmed = true

	return issueTokenPair(user)
}

// issueTokenPair builds the access + refresh token pair for the given user.
func issueTokenPair(user models.User) (accessToken, refreshToken string, err error) {
	perms, permErr := userPermissionSlice(user.ID)
	if permErr != nil {
		return "", "", permErr
	}

	secret := setting.AppSetting.JWTSecret
	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, AccessTokenTTL)
	if err != nil {
		return "", "", err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}
	return at, rt, nil
}

// userPermissionSlice returns a sorted slice of code_check strings for the user (via roles + direct grants).
func userPermissionSlice(userID uint) ([]string, error) {
	set, err := PermissionCodesForUser(userID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for cc := range set {
		out = append(out, cc)
	}
	sort.Strings(out)
	return out, nil
}

// GetMe returns essential (non-sensitive) profile info for the given user along
// with their current permission codes.
func GetMe(userID uint) (*dto.MeResponse, error) {
	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	perms, err := userPermissionSlice(userID)
	if err != nil {
		return nil, err
	}

	return &dto.MeResponse{
		UserID:         user.ID,
		UserCode:       user.UserCode,
		Email:          user.Email,
		DisplayName:    user.DisplayName,
		AvatarURL:      user.AvatarURL,
		EmailConfirmed: user.EmailConfirmed,
		IsDisabled:     user.IsDisable,
		CreatedAt:      user.CreatedAt.Unix(),
		Permissions:    perms,
	}, nil
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
