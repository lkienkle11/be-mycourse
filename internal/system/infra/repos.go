// Package infra implements the SYSTEM domain repository interfaces using GORM.
package infra

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/system/domain"
)

// --- GORM row types ----------------------------------------------------------

type appConfigRow struct {
	ID                   int    `gorm:"column:id;primaryKey"`
	AppCLISystemPassword string `gorm:"column:app_cli_system_password;not null"`
	AppSystemEnv         string `gorm:"column:app_system_env;not null"`
	AppTokenEnv          string `gorm:"column:app_token_env;not null"`
	UpdatedAt            int64  `gorm:"column:updated_at;not null"`
}

func (appConfigRow) TableName() string { return constants.TableSystemAppConfig }

type privilegedUserRow struct {
	ID             uint   `gorm:"column:id;primaryKey"`
	UsernameSecret string `gorm:"column:username_secret;not null;uniqueIndex"`
	PasswordSecret string `gorm:"column:password_secret;not null"`
	MachineSecret  string `gorm:"column:machine_secret;not null"`
	CreatedAt      int64  `gorm:"column:created_at;not null"`
}

func (privilegedUserRow) TableName() string { return constants.TableSystemPrivilegedUsers }

// permissionSyncRow is a minimal GORM row used only during permission sync.
type permissionSyncRow struct {
	PermissionID   string `gorm:"column:permission_id;primaryKey"`
	PermissionName string `gorm:"column:permission_name;not null"`
	Description    string `gorm:"column:description"`
	CreatedAt      int64  `gorm:"column:created_at;not null"`
	UpdatedAt      int64  `gorm:"column:updated_at;not null"`
}

func (permissionSyncRow) TableName() string { return constants.TableRBACPermissions }

type roleSyncRow struct {
	ID   uint   `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name;not null;uniqueIndex"`
}

func (roleSyncRow) TableName() string { return constants.TableRBACRoles }

type rolePermissionSyncRow struct {
	RoleID       uint   `gorm:"column:role_id"`
	PermissionID string `gorm:"column:permission_id"`
}

func (rolePermissionSyncRow) TableName() string { return constants.TableRBACRolePermissions }

// --- GormAppConfigRepository -------------------------------------------------

// GormAppConfigRepository implements domain.AppConfigRepository.
type GormAppConfigRepository struct{ db *gorm.DB }

func NewGormAppConfigRepository(db *gorm.DB) *GormAppConfigRepository {
	return &GormAppConfigRepository{db: db}
}

func (r *GormAppConfigRepository) Get(ctx context.Context) (*domain.AppConfig, error) {
	var row appConfigRow
	if err := r.db.WithContext(ctx).Where("id = ?", 1).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrSystemAppConfigMissing
		}
		return nil, err
	}
	return &domain.AppConfig{
		ID:                   row.ID,
		AppCLISystemPassword: row.AppCLISystemPassword,
		AppSystemEnv:         row.AppSystemEnv,
		AppTokenEnv:          row.AppTokenEnv,
		UpdatedAt:            row.UpdatedAt,
	}, nil
}

// --- GormPrivilegedUserRepository --------------------------------------------

// GormPrivilegedUserRepository implements domain.PrivilegedUserRepository.
type GormPrivilegedUserRepository struct{ db *gorm.DB }

func NewGormPrivilegedUserRepository(db *gorm.DB) *GormPrivilegedUserRepository {
	return &GormPrivilegedUserRepository{db: db}
}

func (r *GormPrivilegedUserRepository) Create(ctx context.Context, u *domain.PrivilegedUser) error {
	row := &privilegedUserRow{
		UsernameSecret: u.UsernameSecret,
		PasswordSecret: u.PasswordSecret,
		MachineSecret:  u.MachineSecret,
	}
	gormx.TouchCreatedUpdated(&row.CreatedAt, nil)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	u.ID = row.ID
	u.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormPrivilegedUserRepository) FindByCredentials(ctx context.Context, usernameSecret, passwordSecret string) (*domain.PrivilegedUser, error) {
	var row privilegedUserRow
	err := r.db.WithContext(ctx).
		Where("username_secret = ? AND password_secret = ?", usernameSecret, passwordSecret).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.PrivilegedUser{
		ID:             row.ID,
		UsernameSecret: row.UsernameSecret,
		PasswordSecret: row.PasswordSecret,
		MachineSecret:  row.MachineSecret,
		CreatedAt:      row.CreatedAt,
	}, nil
}

// --- GormPermissionSyncer ----------------------------------------------------

// GormPermissionSyncer implements domain.PermissionSyncer.
type GormPermissionSyncer struct{ db *gorm.DB }

func NewGormPermissionSyncer(db *gorm.DB) *GormPermissionSyncer {
	return &GormPermissionSyncer{db: db}
}

func (s *GormPermissionSyncer) SyncPermissionsFromCatalog(ctx context.Context, entries []domain.PermissionCatalogEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, e := range entries {
			var row permissionSyncRow
			err := tx.Where("permission_id = ?", e.PermissionID).First(&row).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row = permissionSyncRow{
					PermissionID:   e.PermissionID,
					PermissionName: e.PermissionName,
					Description:    e.Description,
				}
				gormx.TouchCreatedUpdated(&row.CreatedAt, &row.UpdatedAt)
				if createErr := tx.Create(&row).Error; createErr != nil {
					return createErr
				}
				continue
			}
			if err != nil {
				return err
			}
			row.PermissionName = e.PermissionName
			if e.Description != "" {
				row.Description = e.Description
			}
			gormx.TouchUpdated(&row.UpdatedAt)
			if saveErr := tx.Save(&row).Error; saveErr != nil {
				return saveErr
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// --- GormRolePermissionSyncer ------------------------------------------------

// GormRolePermissionSyncer implements domain.RolePermissionSyncer.
type GormRolePermissionSyncer struct{ db *gorm.DB }

func NewGormRolePermissionSyncer(db *gorm.DB) *GormRolePermissionSyncer {
	return &GormRolePermissionSyncer{db: db}
}

func (s *GormRolePermissionSyncer) SyncRolePermissionsFromCatalog(ctx context.Context, pairs []domain.RolePermissionPair) (int, error) {
	if len(pairs) == 0 {
		return 0, nil
	}
	inserted := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", constants.TableRBACRolePermissions)).Error; err != nil {
			return err
		}
		for _, pair := range pairs {
			var role roleSyncRow
			if err := tx.Where("name = ?", pair.RoleName).First(&role).Error; err != nil {
				return fmt.Errorf("role %q: %w", pair.RoleName, err)
			}
			row := rolePermissionSyncRow{RoleID: role.ID, PermissionID: pair.PermID}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("role %q perm %q: %w", pair.RoleName, pair.PermID, err)
			}
			inserted++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}
