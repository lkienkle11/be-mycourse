package infra

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"mycourse-io-be/internal/auth/domain"
	sharedErrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/timex"
)

// GormOAuthIdentityRepository implements domain.OAuthIdentityRepository using GORM.
type GormOAuthIdentityRepository struct {
	db *gorm.DB
}

func NewGormOAuthIdentityRepository(db *gorm.DB) *GormOAuthIdentityRepository {
	return &GormOAuthIdentityRepository{db: db}
}

// FindByProviderSub returns the identity for (provider, sub) or (nil, nil) when absent.
func (r *GormOAuthIdentityRepository) FindByProviderSub(ctx context.Context, provider domain.OAuthProvider, providerSub string) (*domain.UserOAuthIdentity, error) {
	var row oauthIdentityRow
	err := gormx.FirstWhere(ctx, r.db, &row, "provider = ? AND provider_sub = ?", string(provider), providerSub)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toOAuthIdentityDomain(&row), nil
}

func (r *GormOAuthIdentityRepository) ListByUserID(ctx context.Context, userID string) ([]domain.UserOAuthIdentity, error) {
	var rows []oauthIdentityRow
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserOAuthIdentity, 0, len(rows))
	for i := range rows {
		out = append(out, *toOAuthIdentityDomain(&rows[i]))
	}
	return out, nil
}

func (r *GormOAuthIdentityRepository) Create(ctx context.Context, identity *domain.UserOAuthIdentity) error {
	return r.CreateWithDB(ctx, r.db, identity)
}

// CreateWithDB inserts an identity using the given DB handle (supports transactions).
func (r *GormOAuthIdentityRepository) CreateWithDB(ctx context.Context, db *gorm.DB, identity *domain.UserOAuthIdentity) error {
	return r.createWithDB(ctx, db, identity)
}

// createWithDB inserts using the given handle (supports transactions from the application layer).
func (r *GormOAuthIdentityRepository) createWithDB(ctx context.Context, db *gorm.DB, identity *domain.UserOAuthIdentity) error {
	row := toOAuthIdentityRow(identity)
	gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
	if row.LinkedAt == 0 {
		row.LinkedAt = row.CreatedAt
	}
	return gormx.CreateAndThen(ctx, db, row, func() {
		identity.ID = row.ID
		identity.LinkedAt = row.LinkedAt
		identity.CreatedAt = row.CreatedAt
		identity.UpdatedAt = row.UpdatedAt
	})
}

func (r *GormOAuthIdentityRepository) UpdateLastLogin(ctx context.Context, id string, at int64) error {
	return r.db.WithContext(ctx).Model(&oauthIdentityRow{}).
		Where("id = ?", id).
		Updates(map[string]any{"last_login_at": at, "updated_at": timex.NowUnix()}).Error
}
