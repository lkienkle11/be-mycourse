package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/constants"
	sharedErrors "mycourse-io-be/internal/shared/errors"
)

// GormUserRepository implements domain.UserRepository using GORM.
type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) FindByUserCode(ctx context.Context, userCode string) (*domain.User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).Where("user_code = ? AND deleted_at IS NULL", userCode).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) FindByConfirmationToken(ctx context.Context, token string) (*domain.User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).Where("confirmation_token = ? AND deleted_at IS NULL", token).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) Create(ctx context.Context, u *domain.User) error {
	row := toUserRow(u)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	u.ID = row.ID
	u.CreatedAt = row.CreatedAt
	u.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormUserRepository) Save(ctx context.Context, u *domain.User) error {
	row := toUserRow(u)
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormUserRepository) UpdateDisplayName(ctx context.Context, userID uint, displayName string) error {
	return r.db.WithContext(ctx).Model(&userRow{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("display_name", displayName).Error
}

func (r *GormUserRepository) UpdateAvatar(ctx context.Context, userID uint, avatarFileID *string) error {
	return r.db.WithContext(ctx).Model(&userRow{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("avatar_file_id", avatarFileID).Error
}

func (r *GormUserRepository) SoftDelete(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Delete(&userRow{}, userID).Error
}

// GormRefreshSessionRepository implements domain.RefreshSessionRepository using GORM.
type GormRefreshSessionRepository struct {
	db *gorm.DB
}

func NewGormRefreshSessionRepository(db *gorm.DB) *GormRefreshSessionRepository {
	return &GormRefreshSessionRepository{db: db}
}

func (r *GormRefreshSessionRepository) LoadSessions(ctx context.Context, userID uint) (domain.RefreshTokenSessionMap, error) {
	var row struct {
		RefreshTokenSession RefreshTokenSessionMap `gorm:"type:jsonb"`
	}
	q := fmt.Sprintf("SELECT refresh_token_session FROM %s WHERE id = ? AND deleted_at IS NULL", constants.TableAppUsers)
	if err := r.db.WithContext(ctx).Raw(q, userID).Scan(&row).Error; err != nil {
		return nil, err
	}
	return toDomainSessionMap(row.RefreshTokenSession), nil
}

func (r *GormRefreshSessionRepository) SaveSessions(ctx context.Context, userID uint, sessions domain.RefreshTokenSessionMap) error {
	data, err := json.Marshal(sessions)
	if err != nil {
		return err
	}
	q := fmt.Sprintf(`UPDATE %s SET refresh_token_session = ?::jsonb, updated_at = NOW() WHERE id = ?`, constants.TableAppUsers)
	return r.db.WithContext(ctx).Exec(q, string(data), userID).Error
}

func (r *GormRefreshSessionRepository) AddSession(ctx context.Context, userID uint, sessionStr string, entry domain.RefreshSessionEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			RefreshTokenSession RefreshTokenSessionMap `gorm:"type:jsonb"`
		}
		q := fmt.Sprintf("SELECT refresh_token_session FROM %s WHERE id = ? AND deleted_at IS NULL", constants.TableAppUsers)
		if err := tx.Raw(q, userID).Scan(&row).Error; err != nil {
			return err
		}
		sessions := toDomainSessionMap(row.RefreshTokenSession)
		if sessions == nil {
			sessions = make(domain.RefreshTokenSessionMap)
		}
		// Enforce max sessions cap
		if _, exists := sessions[sessionStr]; !exists && len(sessions) >= MaxActiveSessions {
			delete(sessions, pickOldestSessionKey(sessions))
		}
		sessions[sessionStr] = entry

		data, err := json.Marshal(sessions)
		if err != nil {
			return err
		}
		uq := fmt.Sprintf(`UPDATE %s SET refresh_token_session = ?::jsonb, updated_at = NOW() WHERE id = ?`, constants.TableAppUsers)
		return tx.Exec(uq, string(data), userID).Error
	})
}

func (r *GormRefreshSessionRepository) SaveSession(ctx context.Context, userID uint, sessionStr string, entry domain.RefreshSessionEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	q := fmt.Sprintf(`UPDATE %s SET refresh_token_session = jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, true), updated_at = NOW() WHERE id = ? AND deleted_at IS NULL`, constants.TableAppUsers)
	return r.db.WithContext(ctx).Exec(q, sessionStr, string(data), userID).Error
}

func (r *GormRefreshSessionRepository) IncrementEmailSendCount(ctx context.Context, userID uint) (int, error) {
	var count int
	q := fmt.Sprintf(`UPDATE %s SET registration_email_send_total = registration_email_send_total + 1 WHERE id = ? AND deleted_at IS NULL RETURNING registration_email_send_total`, constants.TableAppUsers)
	if err := r.db.WithContext(ctx).Raw(q, userID).Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func pickOldestSessionKey(sessions domain.RefreshTokenSessionMap) string {
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
