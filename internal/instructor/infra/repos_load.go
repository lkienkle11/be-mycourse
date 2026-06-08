package infra

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
)

type identityProjection struct {
	FullName     string `gorm:"column:full_name"`
	AvatarFileID string `gorm:"column:avatar_file_id"`
}

func loadActiveRow(ctx context.Context, db *gorm.DB, dest any, query string, args ...any) error {
	return activeScope(db.WithContext(ctx)).Where(query, args...).First(dest).Error
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
		Row TRow `gorm:"embedded"`
		identityProjection
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
	selectSQL := fmt.Sprintf(
		"%s.*, COALESCE(u.display_name, '') AS full_name, COALESCE(u.avatar_file_id::text, '') AS avatar_file_id",
		alias,
	)
	return activeScopeAlias(db.WithContext(ctx), alias).Table(tableName+" "+alias).
		Select(selectSQL).
		Joins("LEFT JOIN "+constants.TableAppUsers+" u ON u.id = "+alias+".user_id AND u.deleted_at IS NULL").
		Where(query, args...).
		First(dest).Error
}

func mapApplicationWithIdentity(row *applicationRow, identity identityProjection) domain.Application {
	out := appRowToDomain(row)
	out.FullName = identity.FullName
	out.AvatarFileID = identity.AvatarFileID
	return out
}

func mapProfileWithIdentity(row *profileRow, identity identityProjection) domain.Profile {
	out := profileRowToDomain(row)
	out.FullName = identity.FullName
	out.AvatarFileID = identity.AvatarFileID
	return out
}
