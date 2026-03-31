package models

import (
	"time"

	"mycourse-io-be/dbschema"
)

// Permission is the smallest authorization unit. Code is the stable key; CodeCheck is used for runtime checks (RequirePermission, /me/permissions).
type Permission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:128;not null" json:"code"`
	CodeCheck   string    `gorm:"uniqueIndex;size:128;not null" json:"code_check"`
	Description string    `gorm:"size:512" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Permission) TableName() string { return dbschema.RBAC.Permissions() }

// Role groups permissions; users receive roles via user_roles and may get extra permissions via user_permissions.
type Role struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Name        string        `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Description string        `gorm:"size:512" json:"description"`
	Permissions []Permission  `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (Role) TableName() string { return dbschema.RBAC.Roles() }

// UserRole binds an external user id (e.g. Supabase/JWT sub) to a role.
type UserRole struct {
	UserID string `gorm:"size:128;primaryKey;index:idx_user_roles_user" json:"user_id"`
	RoleID uint   `gorm:"primaryKey" json:"role_id"`
}

func (UserRole) TableName() string { return dbschema.RBAC.UserRoles() }

// UserPermission binds a user to an extra permission (in addition to permissions implied by their roles).
type UserPermission struct {
	UserID       string `gorm:"size:128;primaryKey;index:idx_user_permissions_user" json:"user_id"`
	PermissionID uint   `gorm:"primaryKey" json:"permission_id"`
	Permission   Permission `gorm:"foreignKey:PermissionID" json:"permission,omitempty"`
}

func (UserPermission) TableName() string { return dbschema.RBAC.UserPermissions() }
