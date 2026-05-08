package sqlmodel

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"mycourse-io-be/constants"
	"time"
)

// RefreshSessionEntry holds metadata for a single authenticated device session.
type RefreshSessionEntry struct {
	RefreshTokenUUID    string    `json:"refresh_token_uuid"`
	RememberMe          bool      `json:"remember_me"`
	RefreshTokenExpired time.Time `json:"refresh_token_expired"`
}

// RefreshTokenSessionMap is a JSONB-backed map keyed by session string (128 hex chars).
// It maps session_id → session metadata for each active device session.
type RefreshTokenSessionMap map[string]RefreshSessionEntry

func (m RefreshTokenSessionMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *RefreshTokenSessionMap) Scan(src any) error {
	if src == nil {
		*m = RefreshTokenSessionMap{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf(constants.MsgRefreshSessionUnsupportedType, src)
	}
	if len(b) == 0 || string(b) == "null" {
		*m = RefreshTokenSessionMap{}
		return nil
	}
	out := make(RefreshTokenSessionMap)
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = out
	return nil
}
