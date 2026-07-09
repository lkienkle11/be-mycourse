// Package delivery contains the AUTH HTTP handlers, DTOs, and route registration.
package delivery

// RegisterRequest is the body for POST /api/v1/auth/register.
type RegisterRequest struct {
	Email       string `json:"email"        binding:"required,email"`
	Password    string `json:"password"     binding:"required"`
	DisplayName string `json:"display_name" binding:"required,min=1,max=255"`
	Locale      string `json:"locale"`
}

// LoginRequest is the body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email      string `json:"email"       binding:"required,email"`
	Password   string `json:"password"    binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

// ConfirmEmailRequest is the body for POST /api/v1/auth/confirm.
type ConfirmEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// CSRFTokenResponse is the JSON data for GET /api/v1/auth/csrf.
type CSRFTokenResponse struct {
	CSRFToken string `json:"csrf_token"`
}

// LoginSessionTokensResponse is the JSON data for login, confirm, and refresh success.
type LoginSessionTokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	SessionID    string `json:"session_id"`
}

// GoogleLoginRequest is the body for POST /api/v1/auth/google.
type GoogleLoginRequest struct {
	Code       string `json:"code" binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

// GoogleOneTapRequest is the body for POST /api/v1/auth/google/onetap.
type GoogleOneTapRequest struct {
	Credential string `json:"credential" binding:"required"`
}

// TokenValue returns the Google ID token for One Tap sign-in.
func (r GoogleOneTapRequest) TokenValue() string { return r.Credential }

// GoogleMobileRequest is the body for POST /api/v1/auth/google/mobile.
type GoogleMobileRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// TokenValue returns the Google ID token for native mobile sign-in.
func (r GoogleMobileRequest) TokenValue() string { return r.IDToken }

// XLoginRequest is the body for POST /api/v1/auth/x.
type XLoginRequest struct {
	Code         string `json:"code" binding:"required"`
	CodeVerifier string `json:"code_verifier" binding:"required"`
	RememberMe   bool   `json:"remember_me"`
	EntryPoint   string `json:"entrypoint"`
}

// DiscordLoginRequest is the body for POST /api/v1/auth/discord.
type DiscordLoginRequest struct {
	Code       string `json:"code" binding:"required"`
	RememberMe bool   `json:"remember_me"`
	EntryPoint string `json:"entrypoint"`
}

// UpdateMeRequest is the body for PATCH /api/v1/me.
type UpdateMeRequest struct {
	AvatarFileID *string `json:"avatar_file_id" validate:"omitempty,uuid"`
}

// MeResponse is the response body for GET /api/v1/me.
type MeResponse struct {
	UserID          string   `json:"user_id"`
	UserCode        string   `json:"user_code"`
	Email           string   `json:"email"`
	DisplayName     string   `json:"display_name"`
	AvatarURL       *string  `json:"avatar_url,omitempty"`
	AvatarObjectKey *string  `json:"avatar_object_key,omitempty"`
	EmailConfirmed  bool     `json:"email_confirmed"`
	IsDisabled      bool     `json:"is_disabled"`
	CreatedAt       int64    `json:"created_at"`
	Permissions     []string `json:"permissions"`
}

// MyPermissionsResponse is the data payload for GET /api/v1/me/permissions.
type MyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
}
