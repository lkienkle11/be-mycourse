package dto

// RegisterRequest is the request body for POST /api/v1/auth/register.
type RegisterRequest struct {
	Email       string `json:"email"        binding:"required,email"`
	Password    string `json:"password"     binding:"required"`
	DisplayName string `json:"display_name" binding:"required,min=1,max=255"`
}

// LoginRequest is the request body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email      string `json:"email"       binding:"required,email"`
	Password   string `json:"password"    binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

// MeResponse is the response body for GET /api/v1/me.
// Sensitive fields (hash_password, confirmation_token, etc.) are intentionally omitted.
type MeResponse struct {
	UserID         uint             `json:"user_id"`
	UserCode       string           `json:"user_code"`
	Email          string           `json:"email"`
	DisplayName    string           `json:"display_name"`
	Avatar         *MediaFilePublic `json:"avatar,omitempty"`
	EmailConfirmed bool             `json:"email_confirmed"`
	IsDisabled     bool             `json:"is_disabled"`
	CreatedAt      int64            `json:"created_at"`
	Permissions    []string         `json:"permissions"`
}

// UpdateMeRequest is the body for PATCH /api/v1/me (partial profile update).
type UpdateMeRequest struct {
	AvatarFileID *string `json:"avatar_file_id" validate:"omitempty,uuid"`
}

// LoginSessionTokensResponse is the JSON data for login, email confirm, and refresh success
// (tokens are also written to cookies).
type LoginSessionTokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	SessionID    string `json:"session_id"`
}

// MyPermissionsResponse is the data payload for GET /api/v1/me/permissions.
type MyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
}
