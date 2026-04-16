package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
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
		return fmt.Errorf("RefreshTokenSessionMap: unsupported source type %T", src)
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

// MaxActiveSessions is the maximum number of concurrent device sessions allowed per user.
// When a new session is created and the limit is already reached, the session with the
// earliest RefreshTokenExpired is evicted to make room.
const MaxActiveSessions = 5

// SaveRefreshSession atomically updates a single existing session entry inside
// users.refresh_token_session using PostgreSQL's jsonb_set.
// Use this for in-place rotation (the session key stays the same, only metadata changes).
func SaveRefreshSession(userID uint, sessionStr string, entry RefreshSessionEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return DB.Exec(
		`UPDATE users
		 SET refresh_token_session = jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, true),
		     updated_at = NOW()
		 WHERE id = ? AND deleted_at IS NULL`,
		sessionStr,
		string(data),
		userID,
	).Error
}

// AddRefreshSession adds a brand-new session entry for the user, enforcing the
// MaxActiveSessions cap.  When the cap is reached the session whose
// RefreshTokenExpired is earliest (oldest / soonest-to-expire) is evicted first.
// The entire operation runs inside a transaction to prevent concurrent logins from
// exceeding the limit.
func AddRefreshSession(userID uint, sessionStr string, entry RefreshSessionEntry) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var u User
		if err := tx.Select("id", "refresh_token_session").
			Where("id = ? AND deleted_at IS NULL", userID).
			First(&u).Error; err != nil {
			return err
		}

		sessions := u.RefreshTokenSession
		if sessions == nil {
			sessions = RefreshTokenSessionMap{}
		}

		// Evict oldest session when at the cap (new key not already present).
		if _, exists := sessions[sessionStr]; !exists && len(sessions) >= MaxActiveSessions {
			oldestKey := ""
			var oldestExpiry time.Time
			for k, v := range sessions {
				if oldestKey == "" || v.RefreshTokenExpired.Before(oldestExpiry) {
					oldestKey = k
					oldestExpiry = v.RefreshTokenExpired
				}
			}
			delete(sessions, oldestKey)
		}

		sessions[sessionStr] = entry

		data, err := json.Marshal(sessions)
		if err != nil {
			return err
		}
		return tx.Exec(
			`UPDATE users SET refresh_token_session = ?::jsonb, updated_at = NOW() WHERE id = ?`,
			string(data),
			userID,
		).Error
	})
}

// User is the application user stored in the custom `users` table.
// UserCode is a UUIDv7 string generated at the application layer before insert.
// It is used as the user_id in user_roles and user_permissions (RBAC tables).
type User struct {
	ID                  uint                   `gorm:"primaryKey;autoIncrement"         json:"id"`
	UserCode            string                 `gorm:"type:uuid;uniqueIndex;not null"   json:"user_code"`
	Email               string                 `gorm:"size:255;uniqueIndex;not null"    json:"email"`
	HashPassword        string                 `gorm:"size:255;not null"                json:"-"`
	DisplayName         string                 `gorm:"size:255;not null;default:''"     json:"display_name"`
	AvatarURL           string                 `gorm:"type:text;not null;default:''"   json:"avatar_url"`
	IsDisable           bool                   `gorm:"not null;default:false"           json:"is_disable"`
	EmailConfirmed      bool                   `gorm:"not null;default:false"           json:"email_confirmed"`
	ConfirmationToken   *string                `gorm:"size:128"                         json:"-"`
	ConfirmationSentAt  *time.Time             `                                        json:"-"`
	RefreshTokenSession RefreshTokenSessionMap `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	CreatedAt           time.Time              `                                        json:"created_at"`
	UpdatedAt           time.Time              `                                        json:"updated_at"`
	DeletedAt           gorm.DeletedAt         `gorm:"index"                            json:"deleted_at,omitempty"`
}

func (User) TableName() string { return "users" }
