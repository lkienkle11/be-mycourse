package models

import (
	"time"

	"mycourse-io-be/dbschema"
)

// Permission is the smallest authorization unit. PermissionID is the stable PK (e.g. P1).
// PermissionName is what JWT / RequirePermission compares (resource:action).
type Permission struct {
	PermissionID   string    `gorm:"column:permission_id;primaryKey;size:10" json:"permission_id"`
	PermissionName string    `gorm:"column:permission_name;uniqueIndex;size:50;not null" json:"permission_name"`
	Description    string    `gorm:"size:512" json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Permission) TableName() string { return dbschema.RBAC.Permissions() }

// RolePermission is the role ↔ permission junction (explicit table for bulk writes).
type RolePermission struct {
	RoleID       uint   `gorm:"column:role_id;primaryKey" json:"role_id"`
	PermissionID string `gorm:"column:permission_id;primaryKey;size:10" json:"permission_id"`
}

func (RolePermission) TableName() string { return dbschema.RBAC.RolePermissions() }

// Role groups permissions; users receive roles via user_roles and may get extra permissions via user_permissions.
type Role struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Name        string       `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Description string       `gorm:"size:512" json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions;joinForeignKey:role_id;joinReferences:permission_id" json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (Role) TableName() string { return dbschema.RBAC.Roles() }

// UserRole binds a user (users.id) to a role.
type UserRole struct {
	UserID uint `gorm:"primaryKey;index:idx_user_roles_user" json:"user_id"`
	RoleID uint `gorm:"primaryKey" json:"role_id"`
}

func (UserRole) TableName() string { return dbschema.RBAC.UserRoles() }

// UserPermission binds a user (users.id) to an extra permission (in addition to permissions implied by their roles).
type UserPermission struct {
	UserID       uint       `gorm:"primaryKey;index:idx_user_permissions_user" json:"user_id"`
	PermissionID string     `gorm:"primaryKey;size:10" json:"permission_id"`
	Permission   Permission `gorm:"foreignKey:PermissionID;references:PermissionID" json:"permission,omitempty"`
}

func (UserPermission) TableName() string { return dbschema.RBAC.UserPermissions() }
