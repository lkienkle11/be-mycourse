package infra

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/timex"
)

type identityProjection struct {
	FullName       string `gorm:"column:full_name"`
	Slug           string `gorm:"column:slug"`
	Email          string `gorm:"column:email"`
	Phone          string `gorm:"column:phone"`
	AvatarFileID   string `gorm:"column:avatar_file_id"`
	IsDisabled     bool   `gorm:"column:is_disabled"`
	EmailConfirmed bool   `gorm:"column:email_confirmed"`
	BannedUntil    *int64 `gorm:"column:banned_until"`
	IsBanned       bool   `gorm:"column:is_banned"`
}

// identityColumnsSQL is the single user-identity projection (aliased `u`)
// shared by every instructor query that embeds identityProjection: the by-id
// loaders and the application/profile list queries. now is the unix time used
// to evaluate the ban window.
func identityColumnsSQL(now int64) string {
	return fmt.Sprintf(
		"COALESCE(u.display_name, '') AS full_name, COALESCE(u.slug, '') AS slug, COALESCE(u.email, '') AS email, COALESCE(u.phone, '') AS phone, COALESCE(u.avatar_file_id::text, '') AS avatar_file_id, COALESCE(u.is_disable, FALSE) AS is_disabled, COALESCE(u.email_confirmed, FALSE) AS email_confirmed, u.banned_until AS banned_until, (u.banned_until IS NOT NULL AND u.banned_until > %d) AS is_banned",
		now,
	)
}

func loadApplicationRow(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.Application, error) {
	return loadEntityWithIdentity(
		ctx, db, constants.TableInstructorApplications, "ia", query, args, mapApplicationWithIdentity,
	)
}

func loadProfileRow(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.Profile, error) {
	return loadEntityWithIdentity(
		ctx, db, constants.TableInstructorProfiles, "ip", query, args, mapProfileWithIdentity,
	)
}

func loadEntityWithIdentity[TRow any, TDomain any](
	ctx context.Context,
	db *gorm.DB,
	tableName string,
	alias string,
	query string,
	args []any,
	mapFn func(*TRow, identityProjection) TDomain,
) (*TDomain, error) {
	type rowWithIdentity struct {
		Row                TRow `gorm:"embedded"`
		identityProjection `gorm:"embedded"`
	}

	var row rowWithIdentity
	if err := loadRowWithIdentity(ctx, db, tableName, alias, query, args, &row); err != nil {
		return nil, mapNotFound(err)
	}
	out := mapFn(&row.Row, row.identityProjection)
	return &out, nil
}

func loadRowWithIdentity(
	ctx context.Context,
	db *gorm.DB,
	tableName string,
	alias string,
	query string,
	args []any,
	dest any,
) error {
	selectSQL := alias + ".*, " + identityColumnsSQL(timex.NowUnix())
	return activeScopeAlias(db.WithContext(ctx), alias).Table(tableName+" "+alias).
		Select(selectSQL).
		Joins("LEFT JOIN "+constants.TableAppUsers+" u ON u.id = "+alias+".user_id AND u.deleted_at IS NULL").
		Where(query, args...).
		First(dest).Error
}

func mapApplicationWithIdentity(row *applicationRow, identity identityProjection) domain.Application {
	out := appRowToDomain(row)
	out.FullName = identity.FullName
	out.DisplayName = identity.FullName
	out.Slug = identity.Slug
	out.Email = identity.Email
	out.Phone = identity.Phone
	out.AvatarFileID = identity.AvatarFileID
	out.IsDisabled = identity.IsDisabled
	out.EmailConfirmed = identity.EmailConfirmed
	out.BannedUntil = identity.BannedUntil
	out.IsBanned = identity.IsBanned
	return out
}

func mapProfileWithIdentity(row *profileRow, identity identityProjection) domain.Profile {
	out := profileRowToDomain(row)
	out.FullName = identity.FullName
	out.Slug = identity.Slug
	out.Email = identity.Email
	out.AvatarFileID = identity.AvatarFileID
	out.IsDisabled = identity.IsDisabled
	out.EmailConfirmed = identity.EmailConfirmed
	return out
}
