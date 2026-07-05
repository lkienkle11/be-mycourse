package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
)

// syncUserRowTimestamps copies persisted row keys onto the domain user after insert.
func syncUserRowTimestamps(u *domain.User, row *userRow) {
	u.ID = row.ID
	u.CreatedAt = row.CreatedAt
	u.UpdatedAt = row.UpdatedAt
}

// GormUserRepository implements domain.UserRepository using GORM.
type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var row userRow
	if err := findActiveUserWhere(ctx, r.db, &row, "id = ?", id); err != nil {
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var row userRow
	if err := findActiveUserWhere(ctx, r.db, &row, "email = ?", email); err != nil {
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) FindByUserCode(ctx context.Context, userCode string) (*domain.User, error) {
	var row userRow
	if err := findActiveUserWhere(ctx, r.db, &row, "user_code = ?", userCode); err != nil {
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) FindByConfirmationToken(ctx context.Context, token string) (*domain.User, error) {
	var row userRow
	if err := findActiveUserWhere(ctx, r.db, &row, "confirmation_token = ?", token); err != nil {
		return nil, err
	}
	return toUserDomain(&row), nil
}

func (r *GormUserRepository) Create(ctx context.Context, u *domain.User) error {
	row := toUserRow(u)
	gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
	return gormx.CreateAndThen(ctx, r.db, row, func() { syncUserRowTimestamps(u, row) })
}

func (r *GormUserRepository) Save(ctx context.Context, u *domain.User) error {
	return r.SaveWithDB(ctx, r.db, u)
}

// SaveWithDB persists user fields using the given DB handle (supports transactions).
func (r *GormUserRepository) SaveWithDB(ctx context.Context, db *gorm.DB, u *domain.User) error {
	row := toUserRow(u)
	gormx.TouchUpdated(&row.UpdatedAt)
	u.UpdatedAt = row.UpdatedAt
	return db.WithContext(ctx).Save(row).Error
}

func (r *GormUserRepository) UpdateDisplayName(ctx context.Context, userID string, displayName string) error {
	return r.db.WithContext(ctx).Model(&userRow{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("display_name", displayName).Error
}

func (r *GormUserRepository) UpdateAvatar(ctx context.Context, userID string, avatarFileID *string) error {
	return r.db.WithContext(ctx).Model(&userRow{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("avatar_file_id", avatarFileID).Error
}

func (r *GormUserRepository) SoftDelete(ctx context.Context, userID string) error {
	return gormx.SoftDeleteWithAudit(ctx, r.db, &userRow{}, "id = ? AND deleted_at IS NULL", userID)
}

func (r *GormUserRepository) HardDelete(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Delete(&userRow{}, userID).Error
}

// GormRefreshSessionRepository implements domain.RefreshSessionRepository using GORM.
type GormRefreshSessionRepository struct {
	db *gorm.DB
}

func NewGormRefreshSessionRepository(db *gorm.DB) *GormRefreshSessionRepository {
	return &GormRefreshSessionRepository{db: db}
}

func (r *GormRefreshSessionRepository) LoadSessions(ctx context.Context, userID string) (domain.RefreshTokenSessionMap, error) {
	var row struct {
		RefreshTokenSession RefreshTokenSessionMap `gorm:"type:jsonb"`
	}
	q := fmt.Sprintf("SELECT refresh_token_session FROM %s WHERE id = ? AND deleted_at IS NULL", constants.TableAppUsers)
	if err := r.db.WithContext(ctx).Raw(q, userID).Scan(&row).Error; err != nil {
		return nil, err
	}
	return toDomainSessionMap(row.RefreshTokenSession), nil
}

func (r *GormRefreshSessionRepository) SaveSessions(ctx context.Context, userID string, sessions domain.RefreshTokenSessionMap) error {
	data, err := json.Marshal(sessions)
	if err != nil {
		return err
	}
	q := fmt.Sprintf(`UPDATE %s SET refresh_token_session = ?::jsonb, updated_at = ? WHERE id = ?`, constants.TableAppUsers)
	return r.db.WithContext(ctx).Exec(q, string(data), timex.NowUnix(), userID).Error
}

func (r *GormRefreshSessionRepository) AddSession(ctx context.Context, userID string, sessionStr string, entry domain.RefreshSessionEntry) error {
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
		if _, exists := sessions[sessionStr]; !exists && len(sessions) >= MaxActiveSessions {
			delete(sessions, pickOldestSessionKey(sessions))
		}
		sessions[sessionStr] = entry

		data, err := json.Marshal(sessions)
		if err != nil {
			return err
		}
		uq := fmt.Sprintf(`UPDATE %s SET refresh_token_session = ?::jsonb, updated_at = ? WHERE id = ?`, constants.TableAppUsers)
		return tx.Exec(uq, string(data), timex.NowUnix(), userID).Error
	})
}

func (r *GormRefreshSessionRepository) RemoveSession(ctx context.Context, userID string, sessionStr string) error {
	sessions, err := r.LoadSessions(ctx, userID)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		return nil
	}
	if _, ok := sessions[sessionStr]; !ok {
		return nil
	}
	delete(sessions, sessionStr)
	return r.SaveSessions(ctx, userID, sessions)
}

func (r *GormRefreshSessionRepository) SaveSession(ctx context.Context, userID string, sessionStr string, entry domain.RefreshSessionEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	q := fmt.Sprintf(`UPDATE %s SET refresh_token_session = jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, true), updated_at = ? WHERE id = ? AND deleted_at IS NULL`, constants.TableAppUsers)
	return r.db.WithContext(ctx).Exec(q, sessionStr, string(data), timex.NowUnix(), userID).Error
}

func (r *GormRefreshSessionRepository) IncrementEmailSendCount(ctx context.Context, userID string) (int, error) {
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
