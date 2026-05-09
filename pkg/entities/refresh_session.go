package entities

import "time"

// RefreshSessionEntry holds metadata for a single authenticated device session.
type RefreshSessionEntry struct {
	RefreshTokenUUID    string    `json:"refresh_token_uuid"`
	RememberMe          bool      `json:"remember_me"`
	RefreshTokenExpired time.Time `json:"refresh_token_expired"`
}

// RefreshTokenSessionMap is the in-memory shape of users.refresh_token_session JSONB
// (session id → metadata). Database Valuer/Scanner lives on models.RefreshTokenSessionJSONB.
type RefreshTokenSessionMap map[string]RefreshSessionEntry
