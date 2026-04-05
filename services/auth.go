package services

import (
	"errors"
	"sort"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/brevo"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/pkg/token"
)

// Auth token lifetimes — exported so the HTTP layer can derive cookie MaxAge from the same value.
const (
	AccessTokenTTL       = 15 * time.Minute
	RefreshTokenTTL      = 30 * 24 * time.Hour // default / non-remember-me initial TTL
	RememberMeRefreshTTL = 14 * 24 * time.Hour // remember-me: renewed to this on every rotation
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
	ErrInvalidSession      = errors.New("invalid session")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
)

// TokenPairResult carries all token issuance output needed by the HTTP layer.
type TokenPairResult struct {
	AccessToken  string
	RefreshToken string
	// SessionStr is the 128-char hex string that identifies this session.
	// It is delivered to the client via the session_id HttpOnly cookie.
	SessionStr string
	// RefreshTTL is the lifetime of the newly issued refresh token.
	// The HTTP layer uses this to compute the correct cookie MaxAge.
	RefreshTTL time.Duration
}

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

// Login validates credentials and returns a signed token pair plus a session string.
// rememberMe controls whether subsequent token rotations extend the refresh TTL (14 days)
// or preserve the remaining lifetime of the previous token.
func Login(email, password string, rememberMe bool) (TokenPairResult, error) {
	var user models.User
	if dbErr := models.DB.Where("email = ?", email).First(&user).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return TokenPairResult{}, ErrInvalidCredentials
		}
		return TokenPairResult{}, dbErr
	}

	if user.IsDisable {
		return TokenPairResult{}, ErrUserDisabled
	}
	if !user.EmailConfirmed {
		return TokenPairResult{}, ErrEmailNotConfirmed
	}
	if bcrypt.CompareHashAndPassword([]byte(user.HashPassword), []byte(password)) != nil {
		return TokenPairResult{}, ErrInvalidCredentials
	}

	refreshTTL := RefreshTokenTTL
	if rememberMe {
		refreshTTL = RememberMeRefreshTTL
	}
	return issueTokenPair(user, rememberMe, refreshTTL)
}

// ConfirmEmail marks the user's email as confirmed and returns a token pair.
// remember_me is always false for email-confirmation sessions.
func ConfirmEmail(confirmToken string) (TokenPairResult, error) {
	var user models.User
	if dbErr := models.DB.Where("confirmation_token = ?", confirmToken).First(&user).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return TokenPairResult{}, ErrInvalidConfirmToken
		}
		return TokenPairResult{}, dbErr
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
		return TokenPairResult{}, dbErr
	}
	user.EmailConfirmed = true
	user.ConfirmationToken = nil

	return issueTokenPair(user, false, RefreshTokenTTL)
}

// RefreshSession rotates the token pair for an existing session identified by sessionStr.
// It parses the refresh token (ignoring JWT expiry — the DB record is authoritative),
// verifies the session entry, then issues fresh access + refresh tokens while keeping
// the same session string (the client's session_id cookie value is unchanged).
//
// TTL rules on rotation:
//   - remember_me=true  → new refresh TTL is always RememberMeRefreshTTL (14 days from now)
//   - remember_me=false → new refresh TTL equals the remaining lifetime of the old token
func RefreshSession(sessionStr, refreshTokenStr string) (TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret

	refreshClaims, err := token.ParseRefreshIgnoreExpiry(secret, refreshTokenStr)
	if err != nil {
		return TokenPairResult{}, ErrInvalidSession
	}

	var user models.User
	if dbErr := models.DB.First(&user, refreshClaims.UserID).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return TokenPairResult{}, ErrUserNotFound
		}
		return TokenPairResult{}, dbErr
	}

	if user.IsDisable {
		return TokenPairResult{}, ErrUserDisabled
	}

	entry, ok := user.RefreshTokenSession[sessionStr]
	if !ok || entry.RefreshTokenUUID != refreshClaims.UUID {
		return TokenPairResult{}, ErrInvalidSession
	}

	if time.Now().After(entry.RefreshTokenExpired) {
		return TokenPairResult{}, ErrRefreshTokenExpired
	}

	var newRefreshTTL time.Duration
	if entry.RememberMe {
		newRefreshTTL = RememberMeRefreshTTL
	} else {
		newRefreshTTL = time.Until(entry.RefreshTokenExpired)
		if newRefreshTTL <= 0 {
			return TokenPairResult{}, ErrRefreshTokenExpired
		}
	}

	// Build new tokens — reuse the SAME session string so the client cookie is unchanged.
	newUUID := uuid.New().String()

	perms, permErr := userPermissionSlice(user.ID)
	if permErr != nil {
		return TokenPairResult{}, permErr
	}

	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, AccessTokenTTL)
	if err != nil {
		return TokenPairResult{}, err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, newUUID, newRefreshTTL)
	if err != nil {
		return TokenPairResult{}, err
	}

	// Update the existing entry in-place (same key → session count unchanged).
	updatedEntry := models.RefreshSessionEntry{
		RefreshTokenUUID:    newUUID,
		RememberMe:          entry.RememberMe,
		RefreshTokenExpired: time.Now().Add(newRefreshTTL),
	}
	if saveErr := models.SaveRefreshSession(user.ID, sessionStr, updatedEntry); saveErr != nil {
		return TokenPairResult{}, saveErr
	}

	return TokenPairResult{
		AccessToken:  at,
		RefreshToken: rt,
		SessionStr:   sessionStr, // unchanged — client cookie stays the same
		RefreshTTL:   newRefreshTTL,
	}, nil
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

	perms, permErr := userPermissionSlice(user.ID)
	if permErr != nil {
		return TokenPairResult{}, permErr
	}

	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, AccessTokenTTL)
	if err != nil {
		return TokenPairResult{}, err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, sessionUUID, refreshTTL)
	if err != nil {
		return TokenPairResult{}, err
	}

	entry := models.RefreshSessionEntry{
		RefreshTokenUUID:    sessionUUID,
		RememberMe:          rememberMe,
		RefreshTokenExpired: time.Now().Add(refreshTTL),
	}
	if saveErr := models.AddRefreshSession(user.ID, sessionStr, entry); saveErr != nil {
		return TokenPairResult{}, saveErr
	}

	return TokenPairResult{
		AccessToken:  at,
		RefreshToken: rt,
		SessionStr:   sessionStr,
		RefreshTTL:   refreshTTL,
	}, nil
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
