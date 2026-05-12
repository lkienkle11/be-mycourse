// Package domain contains the RBAC bounded-context core entities and repository interfaces.
package domain

import "time"

// Permission is the smallest authorization unit.
type Permission struct {
	PermissionID   string
	PermissionName string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Role groups permissions; users receive roles via UserRole.
type Role struct {
	ID          uint
	Name        string
	Description string
	Permissions []Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UserRole binds a user to a role.
type UserRole struct {
	UserID uint
	RoleID uint
}

// UserPermission binds a user to an extra permission (beyond role grants).
type UserPermission struct {
	UserID       uint
	PermissionID string
	Permission   Permission
}

// SeedPermission defines a permission for seeding the database.
type SeedPermission struct {
	PermissionID   string
	PermissionName string
	Description    string
}

// --- filter/input types ------------------------------------------------------

// PermissionFilter is used for listing permissions.
type PermissionFilter struct {
	Page     int
	PageSize int
}

// RoleFilter is used for listing roles.
type RoleFilter struct {
	Page            int
	PageSize        int
	WithPermissions bool
}

// CreatePermissionInput carries data for creating a permission.
type CreatePermissionInput struct {
	PermissionID   string
	PermissionName string
	Description    string
}

// UpdatePermissionInput carries data for updating a permission.
type UpdatePermissionInput struct {
	PermissionName *string
	Description    *string
}

// CreateRoleInput carries data for creating a role.
type CreateRoleInput struct {
	Name        string
	Description string
}

// UpdateRoleInput carries data for updating a role.
type UpdateRoleInput struct {
	Name        *string
	Description *string
}
