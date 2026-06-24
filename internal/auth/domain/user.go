// Package domain contains the AUTH bounded-context's pure domain types:
// entities, repository interfaces, and domain errors.  No GORM, no HTTP, no Redis.
package domain

import "time"

// RefreshSessionEntry holds metadata for one authenticated device session.
type RefreshSessionEntry struct {
	RefreshTokenUUID    string    `json:"refresh_token_uuid"`
	RememberMe          bool      `json:"remember_me"`
	RefreshTokenExpired time.Time `json:"refresh_token_expired"`
}

// RefreshTokenSessionMap is the in-memory representation of users.refresh_token_session JSONB
// (session-string → entry). The infra layer provides a JSONB Valuer/Scanner.
type RefreshTokenSessionMap map[string]RefreshSessionEntry

// User is the application user entity persisted in the `users` table.
type User struct {
	ID                         string
	UserCode                   string
	Email                      string
	HashPassword               string
	DisplayName                string
	AvatarFileID               *string
	IsDisable                  bool
	EmailConfirmed             bool
	ConfirmationToken          *string
	ConfirmationSentAt         *time.Time
	RegistrationEmailSendTotal int
	RefreshTokenSession        RefreshTokenSessionMap
	CreatedAt                  int64
	UpdatedAt                  int64
	DeletedAt                  *int64
	BannedUntil                *int64 // Unix seconds when a time-limited ban lifts; nil = not banned
}

// MeProfile is the /me projection used for Redis cache and service layer responses.
type MeProfile struct {
	UserID          string
	UserCode        string
	Email           string
	DisplayName     string
	AvatarFileID    *string // raw file ID stored on user
	AvatarURL       *string // resolved public URL; populated by GetMe when media service is available
	AvatarObjectKey *string // cloud storage object key; populated by GetMe when available
	EmailConfirmed  bool
	IsDisabled      bool
	CreatedAt       int64
	Permissions     []string
}

// TokenPairResult carries the issued access/refresh tokens after login, confirm, or refresh.
type TokenPairResult struct {
	AccessToken  string
	RefreshToken string
	SessionStr   string
	RefreshTTL   time.Duration
	RememberMe   bool
}
