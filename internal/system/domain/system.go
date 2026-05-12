// Package domain contains the SYSTEM bounded-context core entities and repository interfaces.
package domain

import "time"

// AppConfig is the singleton row (id must be 1) holding isolated system secrets.
type AppConfig struct {
	ID                   int
	AppCLISystemPassword string
	AppSystemEnv         string
	AppTokenEnv          string
	UpdatedAt            time.Time
}

// PrivilegedUser stores hashed credentials for system-level operators.
type PrivilegedUser struct {
	ID             uint
	UsernameSecret string
	PasswordSecret string
	CreatedAt      time.Time
}

// PermissionCatalogEntry is a permission entry from the constants catalog.
type PermissionCatalogEntry struct {
	PermissionID   string
	PermissionName string
	Description    string
}

// RolePermissionPair is a role-permission binding from the constants catalog.
type RolePermissionPair struct {
	RoleName string
	PermID   string
}
