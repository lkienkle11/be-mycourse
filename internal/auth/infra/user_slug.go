package infra

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/slug"
	"mycourse-io-be/internal/shared/slug/gormslug"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/uuidx"
)

// userSlugMaxLen is the users.slug column limit shared with the delivery layer.
const userSlugMaxLen = domain.MaxUserSlugLen

// userSlugIndexNames are the unique indexes whose violation means a user slug
// race: the active-users partial index and the global former-slug uniqueness.
var userSlugIndexNames = []string{"uix_users_slug_active", "uq_user_slug_history_slug"}

// userSlugHistoryRow is the GORM model for user_slug_history: one row per
// former slug, reserved permanently so an old link never resolves to another
// person.
type userSlugHistoryRow struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	UserID    string `gorm:"column:user_id;type:uuid;not null"`
	Slug      string `gorm:"column:slug;size:255;not null"`
	CreatedAt int64  `gorm:"column:created_at;not null"`
}

func (userSlugHistoryRow) TableName() string { return constants.TableUserSlugHistory }

func newUserRow() *userRow { return &userRow{} }

func isUserSlugDuplicateKey(err error) bool {
	return gormslug.IsDuplicate(err, userSlugIndexNames...)
}

// userSlugAvailable builds the availability callback for user slugs: a
// candidate is free when no other active user holds it, it is not a reserved
// word, and no other user ever held it. excludeUserID lets a user keep or
// return to their own current/former slug.
func userSlugAvailable(ctx context.Context, db *gorm.DB, excludeUserID *string) slug.AvailableFunc {
	activeFree := gormslug.AvailableIn(ctx, db, newUserRow, excludeUserID)
	return func(candidate string) (bool, error) {
		if slug.IsReserved(candidate) {
			return false, nil
		}
		free, err := activeFree(candidate)
		if err != nil || !free {
			return false, err
		}
		q := db.WithContext(ctx).Model(&userSlugHistoryRow{}).Where("slug = ?", candidate)
		if excludeUserID != nil && *excludeUserID != "" {
			q = q.Where("user_id <> ?", *excludeUserID)
		}
		var count int64
		if err := q.Count(&count).Error; err != nil {
			return false, err
		}
		return count == 0, nil
	}
}

// resolveUserCreateSlug derives the slug for a new user from displayName (no
// manual slug exists on create) through the shared resolver.
func resolveUserCreateSlug(ctx context.Context, db *gorm.DB, displayName string) (string, error) {
	final, _, err := slug.ResolveCreate("", displayName, slug.KindUser, userSlugMaxLen, userSlugAvailable(ctx, db, nil))
	return final, err
}

// ChangeSlug implements domain.UserRepository: resolve (auto-suffix on a taken
// or reserved value), release the user's own matching history row, record the
// slug being left, and write the new slug, all in one transaction with
// savepoint-safe retry on a lost unique-index race.
func (r *GormUserRepository) ChangeSlug(ctx context.Context, userID string, newSlug string) (string, error) {
	var stored string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current userRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "slug").
			Where("id = ? AND deleted_at IS NULL", userID).
			First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return err
		}
		old := slugFromRow(current.Slug)
		if old == newSlug {
			stored = old
			return nil
		}
		return gormx.RetryInSavepoint(tx, "user_slug_change_retry", gormslug.RetryAttempts, isUserSlugDuplicateKey, func() error {
			final, err := slug.ResolveUpdate(newSlug, userSlugMaxLen, userSlugAvailable(ctx, tx, &userID))
			if err != nil {
				return err
			}
			if err := replaceSlugWithHistory(ctx, tx, userID, old, final); err != nil {
				return err
			}
			stored = final
			return nil
		})
	})
	if err != nil {
		return "", err
	}
	return stored, nil
}

// replaceSlugWithHistory writes final as the user's slug and keeps the history
// consistent: a history row equal to final that belongs to this user is
// released (they are returning to a former slug), and the slug being left
// is recorded as former.
func replaceSlugWithHistory(ctx context.Context, tx *gorm.DB, userID, old, final string) error {
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND slug = ?", userID, final).
		Delete(&userSlugHistoryRow{}).Error; err != nil {
		return err
	}
	if old != "" {
		row := &userSlugHistoryRow{ID: uuidx.NewV4(), UserID: userID, Slug: old, CreatedAt: timex.NowUnix()}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
			return err
		}
	}
	return tx.WithContext(ctx).Model(&userRow{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]any{"slug": final, "updated_at": timex.NowUnix()}).Error
}
