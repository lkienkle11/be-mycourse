package application

import (
	"context"

	"mycourse-io-be/internal/auth/domain"
)

// ExternalIdentityInput is the normalized provider profile passed into the generic OAuth core.
type ExternalIdentityInput struct {
	Provider      domain.OAuthProvider
	ProviderSub   string
	Email         string
	EmailVerified bool
	DisplayName   string
	PictureURL    string
	Channel       string
	RememberMe    bool
}

// ProviderPolicy declares provider-specific merge and validation rules for LoginOrCreateFromExternal.
type ProviderPolicy struct {
	RequiresVerifiedEmail     bool
	AllowMergeExistingEmail   bool
	AllowPendingRegisterMerge bool
	SetEmailConfirmedOnCreate bool
	RejectWhenEmailEmpty      bool
}

// GoogleOAuthPolicy is the locked merge policy for Google sign-in.
var GoogleOAuthPolicy = ProviderPolicy{
	RequiresVerifiedEmail:     true,
	AllowMergeExistingEmail:   true,
	AllowPendingRegisterMerge: true,
	SetEmailConfirmedOnCreate: true,
	RejectWhenEmailEmpty:      false,
}

// XOAuthPolicy is the locked merge policy for X sign-in (no auto-merge on email conflict).
var XOAuthPolicy = ProviderPolicy{
	RequiresVerifiedEmail:     false,
	AllowMergeExistingEmail:   false,
	AllowPendingRegisterMerge: false,
	SetEmailConfirmedOnCreate: false,
	RejectWhenEmailEmpty:      true,
}

// DiscordOAuthPolicy is the locked merge policy for Discord sign-in (merge semantics like Google).
var DiscordOAuthPolicy = ProviderPolicy{
	RequiresVerifiedEmail:     true,
	AllowMergeExistingEmail:   true,
	AllowPendingRegisterMerge: true,
	SetEmailConfirmedOnCreate: true,
	RejectWhenEmailEmpty:      true,
}

// OAuthAccountWriter persists OAuth user/identity mutations atomically (implemented in server wiring).
type OAuthAccountWriter interface {
	CreateUserWithIdentityAndLearnerRole(ctx context.Context, user *domain.User, identity *domain.UserOAuthIdentity) error
	LinkIdentityAndUpdateUser(ctx context.Context, user *domain.User, identity *domain.UserOAuthIdentity) error
}

// GoogleOAuthClient verifies Google authorization codes and ID tokens.
type GoogleOAuthClient interface {
	ExchangeCodeAndVerify(ctx context.Context, code string) (ExternalIdentityInput, error)
	VerifyIDToken(ctx context.Context, rawToken string) (ExternalIdentityInput, error)
}

// XOAuthClient exchanges X authorization codes and loads the current user profile.
type XOAuthClient interface {
	ExchangeCodeAndLoadIdentity(ctx context.Context, code, codeVerifier, channel string) (ExternalIdentityInput, error)
}

// DiscordOAuthClient exchanges Discord authorization codes and loads the current user profile.
type DiscordOAuthClient interface {
	ExchangeCodeAndLoadIdentity(ctx context.Context, code, channel string) (ExternalIdentityInput, error)
}
