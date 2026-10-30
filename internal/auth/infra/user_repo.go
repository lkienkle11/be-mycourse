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
	"mycourse-io-be/internal/shared/slug/gormslug"
	"mycourse-io-be/internal/shared/timex"
)

// SQL of the session writers. The guarantees (conditional rotate, row lock, atomic key delete)
// live in these statements, so they are named constants that tests can assert as text.
const (
	sqlRotateSession = "UPDATE " + constants.TableAppUsers +
		" SET refresh_token_session = jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, false), updated_at = ?" +
		" WHERE id = ? AND deleted_at IS NULL AND refresh_token_session -> ?::text ->> 'refresh_token_uuid' = ?"
	sqlSelectSessionsForUpdate = "SELECT refresh_token_session FROM " + constants.TableAppUsers +
		" WHERE id = ? AND deleted_at IS NULL FOR UPDATE"
	sqlStoreSessions = "UPDATE " + constants.TableAppUsers +
		" SET refresh_token_session = ?::jsonb, updated_at = ? WHERE id = ? AND deleted_at IS NULL"
	sqlRemoveSession = "UPDATE " + constants.TableAppUsers +
		" SET refresh_token_session = refresh_token_session - ?::text, updated_at = ?" +
		" WHERE id = ? AND deleted_at IS NULL AND refresh_token_session -> ?::text IS NOT NULL"
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
	return r.CreateWithDB(ctx, r.db, u)
}

// CreateWithDB persists a new user using the given DB handle (supports
// transactions) and assigns its slug in the same statement scope. The insert
// runs inside a transaction (a SAVEPOINT when db is already a transaction, as
// on the OAuth sign-up path) and is retried with a freshly resolved slug when
// it loses a unique-index race, so a slug collision never aborts the caller's
// surrounding transaction nor surfaces as a raw database error.
func (r *GormUserRepository) CreateWithDB(ctx context.Context, db *gorm.DB, u *domain.User) error {
	row := toUserRow(u)
	gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return gormx.RetryInSavepoint(tx, "user_slug_create_retry", gormslug.RetryAttempts, isUserSlugDuplicateKey, func() error {
			final, err := resolveUserCreateSlug(ctx, tx, u.DisplayName)
			if err != nil {
				return err
			}
			row.Slug = &final
			return gormx.CreateAndThen(ctx, tx, row, func() {
				u.Slug = final
				syncUserRowTimestamps(u, row)
			})
		})
	})
}

func (r *GormUserRepository) Save(ctx context.Context, u *domain.User) error {
	return r.SaveWithDB(ctx, r.db, u)
}

// SaveWithDB persists user fields using the given DB handle (supports transactions).
func (r *GormUserRepository) SaveWithDB(ctx context.Context, db *gorm.DB, u *domain.User) error {
	row := toUserRow(u)
	gormx.TouchUpdated(&row.UpdatedAt)
	u.UpdatedAt = row.UpdatedAt
	// slug is owned by CreateWithDB, ChangeSlug and BackfillMissingSlugs, and
	// refresh_token_session by GormRefreshSessionRepository. A full-row save
	// writes neither, so a stale in-memory user can not revert a slug changed
	// meanwhile or erase a session rotated meanwhile.
	return db.WithContext(ctx).Omit("slug", "refresh_token_session").Save(row).Error
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
	return gormx.HardDeleteByID(ctx, r.db, &userRow{}, userID)
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

func (r *GormRefreshSessionRepository) AddSession(ctx context.Context, userID string, sessionStr string, entry domain.RefreshSessionEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			RefreshTokenSession RefreshTokenSessionMap `gorm:"type:jsonb"`
		}
		if err := tx.Raw(sqlSelectSessionsForUpdate, userID).Scan(&row).Error; err != nil {
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
		return tx.Exec(sqlStoreSessions, string(data), timex.NowUnix(), userID).Error
	})
}

// RemoveSession deletes one session key in a single statement, so it can not overwrite a
// concurrent rotation or login. A missing key or user is a no-op (idempotent logout).
func (r *GormRefreshSessionRepository) RemoveSession(ctx context.Context, userID string, sessionStr string) error {
	return r.db.WithContext(ctx).Exec(sqlRemoveSession, sessionStr, timex.NowUnix(), userID, sessionStr).Error
}

// RotateSession replaces the session entry only while its stored refresh_token_uuid equals
// expectedUUID. It returns domain.ErrInvalidSession when no row matched (the session was
// rotated, revoked or the user is gone), so exactly one of several concurrent refreshes wins.
func (r *GormRefreshSessionRepository) RotateSession(ctx context.Context, userID, sessionStr, expectedUUID string, entry domain.RefreshSessionEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	res := r.db.WithContext(ctx).Exec(sqlRotateSession, sessionStr, string(data), timex.NowUnix(), userID, sessionStr, expectedUUID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrInvalidSession
	}
	return nil
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
