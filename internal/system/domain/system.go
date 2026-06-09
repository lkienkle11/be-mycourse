// Package domain contains the SYSTEM bounded-context core entities and repository interfaces.
package domain

// AppConfig is the singleton row (id must be 1) holding isolated system secrets.
type AppConfig struct {
	ID                   int
	AppCLISystemPassword string
	AppSystemEnv         string
	AppTokenEnv          string
	UpdatedAt            int64
}

// PrivilegedUser stores hashed credentials for system-level operators.
type PrivilegedUser struct {
	ID             string
	UsernameSecret string
	PasswordSecret string
	MachineSecret  string
	CreatedAt      int64
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
