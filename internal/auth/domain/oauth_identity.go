package domain

import "context"

// OAuthProvider is a stable lowercase slug identifying an external identity provider.
type OAuthProvider string

const (
	OAuthProviderGoogle  OAuthProvider = "google"
	OAuthProviderX       OAuthProvider = "x"
	OAuthProviderDiscord OAuthProvider = "discord"
)

// OAuth login channels — recorded in identity metadata and used for TTL/session semantics.
const (
	OAuthChannelWebPopupLogin  = "web_popup_login"
	OAuthChannelWebPopupSignup = "web_popup_signup"
	OAuthChannelWebOneTap      = "web_onetap"
	OAuthChannelMobileNative   = "mobile_native"
)

// UserOAuthIdentity is an external identity linked to a MyCourse user.
// Time fields are Unix epoch seconds to align with the users table.
type UserOAuthIdentity struct {
	ID            string
	UserID        string
	Provider      OAuthProvider
	ProviderSub   string
	ProviderEmail *string
	LinkedAt      int64
	LastLoginAt   *int64
	Metadata      map[string]any
	CreatedAt     int64
	UpdatedAt     int64
}

// OAuthIdentityRepository persists external identities. It is intentionally separate
// from UserRepository so user queries do not carry OAuth concerns.
type OAuthIdentityRepository interface {
	FindByProviderSub(ctx context.Context, provider OAuthProvider, providerSub string) (*UserOAuthIdentity, error)
	ListByUserID(ctx context.Context, userID string) ([]UserOAuthIdentity, error)
	Create(ctx context.Context, identity *UserOAuthIdentity) error
	UpdateLastLogin(ctx context.Context, id string, at int64) error
}
