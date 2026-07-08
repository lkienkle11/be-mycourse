package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/uuidx"
)

const oauthIdentityRetryMax = 3

// AttachOAuth wires optional OAuth dependencies after AuthService construction.
func (s *AuthService) AttachOAuth(
	identityRepo domain.OAuthIdentityRepository,
	accountWriter OAuthAccountWriter,
	google GoogleOAuthClient,
	xClient XOAuthClient,
	discord DiscordOAuthClient,
) {
	s.oauthIdentityRepo = identityRepo
	s.oauthAccountWriter = accountWriter
	s.googleOAuth = google
	s.xOAuth = xClient
	s.discordOAuth = discord
}

// GoogleLoginFromCode completes popup Google sign-in from an authorization code.
func (s *AuthService) GoogleLoginFromCode(ctx context.Context, code string, rememberMe bool) (domain.TokenPairResult, error) {
	if s.googleOAuth == nil {
		return domain.TokenPairResult{}, errors.New("google oauth not configured")
	}
	input, err := s.googleOAuth.ExchangeCodeAndVerify(ctx, code)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	input.RememberMe = rememberMe
	return s.LoginOrCreateFromExternal(ctx, input, GoogleOAuthPolicy)
}

// GoogleLoginFromCredential completes One Tap sign-in (always 3-day refresh TTL).
func (s *AuthService) GoogleLoginFromCredential(ctx context.Context, credential string) (domain.TokenPairResult, error) {
	return s.googleLoginFromVerifiedIDToken(ctx, credential, domain.OAuthChannelWebOneTap)
}

// GoogleLoginFromIDToken completes mobile native Google sign-in (always 3-day refresh TTL).
func (s *AuthService) GoogleLoginFromIDToken(ctx context.Context, idToken string) (domain.TokenPairResult, error) {
	return s.googleLoginFromVerifiedIDToken(ctx, idToken, domain.OAuthChannelMobileNative)
}

func (s *AuthService) googleLoginFromVerifiedIDToken(
	ctx context.Context,
	token string,
	channel string,
) (domain.TokenPairResult, error) {
	if s.googleOAuth == nil {
		return domain.TokenPairResult{}, errors.New("google oauth not configured")
	}
	input, err := s.googleOAuth.VerifyIDToken(ctx, token)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	input.Channel = channel
	input.RememberMe = false
	return s.LoginOrCreateFromExternal(ctx, input, GoogleOAuthPolicy)
}

// DiscordLoginFromCode completes Discord OAuth2 sign-in.
func (s *AuthService) DiscordLoginFromCode(ctx context.Context, code, entrypoint string, rememberMe bool) (domain.TokenPairResult, error) {
	if s.discordOAuth == nil {
		return domain.TokenPairResult{}, errors.New("discord oauth not configured")
	}
	channel := domain.OAuthChannelWebPopupLogin
	if entrypoint == "signup" {
		channel = domain.OAuthChannelWebPopupSignup
		rememberMe = false
	}
	input, err := s.discordOAuth.ExchangeCodeAndLoadIdentity(ctx, code, channel)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	input.RememberMe = rememberMe
	return s.LoginOrCreateFromExternal(ctx, input, DiscordOAuthPolicy)
}

// XLoginFromCode completes X OAuth2 PKCE sign-in.
func (s *AuthService) XLoginFromCode(ctx context.Context, code, codeVerifier, entrypoint string, rememberMe bool) (domain.TokenPairResult, error) {
	if s.xOAuth == nil {
		return domain.TokenPairResult{}, errors.New("x oauth not configured")
	}
	channel := domain.OAuthChannelWebPopupLogin
	if entrypoint == "signup" {
		channel = domain.OAuthChannelWebPopupSignup
		rememberMe = false
	}
	input, err := s.xOAuth.ExchangeCodeAndLoadIdentity(ctx, code, codeVerifier, channel)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	input.RememberMe = rememberMe
	return s.LoginOrCreateFromExternal(ctx, input, XOAuthPolicy)
}

// LoginOrCreateFromExternal resolves or creates a user from a verified external identity.
func (s *AuthService) LoginOrCreateFromExternal(
	ctx context.Context,
	input ExternalIdentityInput,
	policy ProviderPolicy,
) (domain.TokenPairResult, error) {
	if s.oauthIdentityRepo == nil || s.oauthAccountWriter == nil {
		return domain.TokenPairResult{}, errors.New("oauth not configured")
	}
	if err := validateExternalIdentityInput(input, policy); err != nil {
		return domain.TokenPairResult{}, err
	}

	var lastConflict error
	for attempt := 0; attempt < oauthIdentityRetryMax; attempt++ {
		user, identityID, err := s.resolveOAuthUser(ctx, input, policy)
		if err != nil {
			if isOAuthUniqueViolation(err) {
				lastConflict = domain.ErrOAuthIdentityConflict
				continue
			}
			return domain.TokenPairResult{}, err
		}

		refreshTTL := domain.RefreshTokenTTL
		if input.RememberMe {
			refreshTTL = domain.RememberMeRefreshTTL
		}
		result, err := s.issueTokenPair(ctx, user, input.RememberMe, refreshTTL)
		if err != nil {
			return domain.TokenPairResult{}, err
		}
		s.delLoginInvalidCache(ctx, normalizeEmail(user.Email))
		s.warmMeCache(ctx, user)
		if identityID != "" {
			_ = s.oauthIdentityRepo.UpdateLastLogin(ctx, identityID, timex.NowUnix())
		}
		return result, nil
	}
	if lastConflict != nil {
		return domain.TokenPairResult{}, lastConflict
	}
	return domain.TokenPairResult{}, domain.ErrOAuthIdentityConflict
}

func validateExternalIdentityInput(input ExternalIdentityInput, policy ProviderPolicy) error {
	if strings.TrimSpace(input.ProviderSub) == "" {
		switch input.Provider {
		case domain.OAuthProviderX:
			return domain.ErrInvalidXCode
		case domain.OAuthProviderDiscord:
			return domain.ErrInvalidDiscordCode
		default:
			return domain.ErrInvalidGoogleCode
		}
	}
	email := strings.TrimSpace(input.Email)
	if policy.RejectWhenEmailEmpty && email == "" {
		switch input.Provider {
		case domain.OAuthProviderX:
			return domain.ErrXEmailUnavailable
		case domain.OAuthProviderDiscord:
			return domain.ErrDiscordEmailUnavailable
		}
	}
	if policy.RequiresVerifiedEmail && (email == "" || !input.EmailVerified) {
		switch input.Provider {
		case domain.OAuthProviderDiscord:
			if email == "" {
				return domain.ErrDiscordEmailUnavailable
			}
			return domain.ErrDiscordEmailNotVerified
		default:
			return domain.ErrGoogleEmailNotVerified
		}
	}
	return nil
}

func (s *AuthService) resolveOAuthUser(
	ctx context.Context,
	input ExternalIdentityInput,
	policy ProviderPolicy,
) (*domain.User, string, error) {
	existingIdentity, err := s.oauthIdentityRepo.FindByProviderSub(ctx, input.Provider, input.ProviderSub)
	if err != nil {
		return nil, "", err
	}
	if existingIdentity != nil {
		return s.loginExistingOAuthIdentity(ctx, existingIdentity)
	}
	return s.resolveOAuthUserByEmail(ctx, input, policy)
}

func (s *AuthService) loginExistingOAuthIdentity(ctx context.Context, identity *domain.UserOAuthIdentity) (*domain.User, string, error) {
	user, err := s.userRepo.FindByID(ctx, identity.UserID)
	if err != nil {
		return nil, "", err
	}
	if err := checkUserAccessible(user, timex.NowUnix()); err != nil {
		return nil, "", err
	}
	return user, identity.ID, nil
}

func (s *AuthService) resolveOAuthUserByEmail(
	ctx context.Context,
	input ExternalIdentityInput,
	policy ProviderPolicy,
) (*domain.User, string, error) {
	normEmail := normalizeEmail(input.Email)
	user, err := s.userRepo.FindByEmail(ctx, normEmail)
	if err == nil {
		return s.linkOAuthToExistingUser(ctx, user, input, policy)
	}
	if !isNotFound(err) {
		return nil, "", err
	}
	return s.createOAuthUser(ctx, input, policy)
}

func (s *AuthService) linkOAuthToExistingUser(
	ctx context.Context,
	user *domain.User,
	input ExternalIdentityInput,
	policy ProviderPolicy,
) (*domain.User, string, error) {
	if err := checkUserAccessible(user, timex.NowUnix()); err != nil {
		return nil, "", err
	}
	if !policy.AllowMergeExistingEmail {
		return nil, "", domain.ErrXAccountLinkRequired
	}
	if user.EmailConfirmed || !policy.AllowPendingRegisterMerge {
		return s.linkIdentityOnly(ctx, user, input)
	}
	return s.mergePendingRegisterWithOAuth(ctx, user, input)
}

func (s *AuthService) linkIdentityOnly(ctx context.Context, user *domain.User, input ExternalIdentityInput) (*domain.User, string, error) {
	identity := newOAuthIdentity(user.ID, input)
	if err := s.oauthAccountWriter.LinkIdentityAndUpdateUser(ctx, user, identity); err != nil {
		return nil, "", err
	}
	return user, identity.ID, nil
}

func (s *AuthService) mergePendingRegisterWithOAuth(ctx context.Context, user *domain.User, input ExternalIdentityInput) (*domain.User, string, error) {
	now := time.Now()
	user.EmailConfirmed = true
	user.PasswordSetAt = &now
	user.ConfirmationToken = nil
	user.RegistrationEmailSendTotal = 0
	if strings.TrimSpace(user.DisplayName) == "" && strings.TrimSpace(input.DisplayName) != "" {
		user.DisplayName = strings.TrimSpace(input.DisplayName)
	}
	identity := newOAuthIdentity(user.ID, input)
	if err := s.oauthAccountWriter.LinkIdentityAndUpdateUser(ctx, user, identity); err != nil {
		return nil, "", err
	}
	return user, identity.ID, nil
}

func (s *AuthService) createOAuthUser(
	ctx context.Context,
	input ExternalIdentityInput,
	policy ProviderPolicy,
) (*domain.User, string, error) {
	hash, err := randomInternalPasswordHash()
	if err != nil {
		return nil, "", err
	}
	uid, err := uuidx.NewV7()
	if err != nil {
		return nil, "", err
	}
	displayName := strings.TrimSpace(input.DisplayName)
	user := &domain.User{
		ID:             uid,
		UserCode:       uuidx.NewULID(),
		Email:          normalizeEmail(input.Email),
		HashPassword:   hash,
		DisplayName:    displayName,
		EmailConfirmed: policy.SetEmailConfirmedOnCreate,
	}
	identity := newOAuthIdentity(uid, input)
	if err := s.oauthAccountWriter.CreateUserWithIdentityAndLearnerRole(ctx, user, identity); err != nil {
		return nil, "", err
	}
	return user, identity.ID, nil
}

func newOAuthIdentity(userID string, input ExternalIdentityInput) *domain.UserOAuthIdentity {
	now := timex.NowUnix()
	meta := map[string]any{
		"channel": input.Channel,
	}
	if input.DisplayName != "" {
		meta["name"] = input.DisplayName
	}
	if input.PictureURL != "" {
		meta["picture"] = input.PictureURL
	}
	var providerEmail *string
	if email := strings.TrimSpace(input.Email); email != "" {
		providerEmail = &email
	}
	id, _ := uuidx.NewV7()
	return &domain.UserOAuthIdentity{
		ID:            id,
		UserID:        userID,
		Provider:      input.Provider,
		ProviderSub:   input.ProviderSub,
		ProviderEmail: providerEmail,
		LinkedAt:      now,
		Metadata:      meta,
	}
}

func randomInternalPasswordHash() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(buf)
	return hashPassword(secret)
}

func isOAuthUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
