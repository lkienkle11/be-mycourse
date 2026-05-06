package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/dbschema"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/sqlmodel"
)

// SaveRefreshSession atomically updates a single existing session entry inside
// users.refresh_token_session using PostgreSQL's jsonb_set.
// Use this for in-place rotation (the session key stays the same, only metadata changes).
func SaveRefreshSession(db *gorm.DB, userID uint, sessionStr string, entry sqlmodel.RefreshSessionEntry) error {
	if db == nil {
		return pkgerrors.ErrNilDatabase
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	q := fmt.Sprintf(`UPDATE %s
		 SET refresh_token_session = jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, true),
		     updated_at = NOW()
		 WHERE id = ? AND deleted_at IS NULL`, dbschema.AppUser.Table())
	return db.Exec(
		q,
		sessionStr,
		string(data),
		userID,
	).Error
}

func pickOldestRefreshSessionKey(sessions sqlmodel.RefreshTokenSessionMap) string {
	oldestKey := ""
	var oldestExpiry time.Time
	first := true
	for k, v := range sessions {
		if first || v.RefreshTokenExpired.Before(oldestExpiry) {
			oldestKey = k
			oldestExpiry = v.RefreshTokenExpired
			first = false
		}
	}
	return oldestKey
}

func mergeNewRefreshSession(sessions sqlmodel.RefreshTokenSessionMap, sessionStr string, entry sqlmodel.RefreshSessionEntry) sqlmodel.RefreshTokenSessionMap {
	if sessions == nil {
		sessions = sqlmodel.RefreshTokenSessionMap{}
	}
	if _, exists := sessions[sessionStr]; !exists && len(sessions) >= constants.MaxActiveSessions {
		delete(sessions, pickOldestRefreshSessionKey(sessions))
	}
	sessions[sessionStr] = entry
	return sessions
}

// AddRefreshSession adds a brand-new session entry for the user, enforcing the
// MaxActiveSessions cap. When the cap is reached the session whose
// RefreshTokenExpired is earliest is evicted first.
// The entire operation runs inside a transaction to prevent concurrent logins from
// exceeding the limit.
func AddRefreshSession(db *gorm.DB, userID uint, sessionStr string, entry sqlmodel.RefreshSessionEntry) error {
	if db == nil {
		return pkgerrors.ErrNilDatabase
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var u models.User
		if err := tx.Select("id", "refresh_token_session").
			Where("id = ? AND deleted_at IS NULL", userID).
			First(&u).Error; err != nil {
			return err
		}
		sessions := mergeNewRefreshSession(u.RefreshTokenSession, sessionStr, entry)
		data, err := json.Marshal(sessions)
		if err != nil {
			return err
		}
		q := fmt.Sprintf(`UPDATE %s SET refresh_token_session = ?::jsonb, updated_at = NOW() WHERE id = ?`,
			dbschema.AppUser.Table())
		return tx.Exec(q, string(data), userID).Error
	})
}
