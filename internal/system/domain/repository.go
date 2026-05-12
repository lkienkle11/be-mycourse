package domain

import "context"

// AppConfigRepository accesses the singleton system_app_config row.
type AppConfigRepository interface {
	Get(ctx context.Context) (*AppConfig, error)
}

// PrivilegedUserRepository manages system privileged-user credentials.
type PrivilegedUserRepository interface {
	Create(ctx context.Context, u *PrivilegedUser) error
	MatchCount(ctx context.Context, usernameSecret, passwordSecret string) (int64, error)
}

// PermissionSyncer syncs permission catalog rows from constants to the database.
type PermissionSyncer interface {
	SyncPermissionsFromCatalog(ctx context.Context, entries []PermissionCatalogEntry) (int, error)
}

// RolePermissionSyncer syncs role-permission pairs from constants to the database.
type RolePermissionSyncer interface {
	SyncRolePermissionsFromCatalog(ctx context.Context, pairs []RolePermissionPair) (int, error)
}
