package entities

import "time"

// RefreshSessionEntry holds metadata for a single authenticated device session.
type RefreshSessionEntry struct {
	RefreshTokenUUID    string    `json:"refresh_token_uuid"`
	RememberMe          bool      `json:"remember_me"`
	RefreshTokenExpired time.Time `json:"refresh_token_expired"`
}

// RefreshTokenSessionMap is the in-memory shape of users.refresh_token_session JSONB
// (session id → metadata). JSONB Valuer/Scanner: **`pkg/gormjsonb/auth`** (`RefreshTokenSessionMap` defined type over `sessionColumnJSONB`).
// From DB model field to this map type, use **`pkg/logic/mapping.ToRefreshTokenSessionEntity`** (Rule 14).
type RefreshTokenSessionMap map[string]RefreshSessionEntry
