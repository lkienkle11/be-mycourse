package domain

import "context"

// PermissionRepository defines persistence for Permission.
type PermissionRepository interface {
	List(ctx context.Context, filter PermissionFilter) ([]Permission, int64, error)
	GetByID(ctx context.Context, permissionID string) (*Permission, error)
	Create(ctx context.Context, p *Permission) error
	Save(ctx context.Context, p *Permission) error
	Delete(ctx context.Context, permissionID string) error
	Upsert(ctx context.Context, p *Permission) error
}

// RoleRepository defines persistence for Role.
type RoleRepository interface {
	List(ctx context.Context, filter RoleFilter) ([]Role, int64, error)
	GetByID(ctx context.Context, id uint, withPermissions bool) (*Role, error)
	Create(ctx context.Context, r *Role) error
	Save(ctx context.Context, r *Role) error
	Delete(ctx context.Context, id uint) error
	AssignPermissions(ctx context.Context, roleID uint, permissionIDs []string) error
	RemovePermissions(ctx context.Context, roleID uint, permissionIDs []string) error
	RemoveAllPermissions(ctx context.Context, roleID uint) error
}

// UserRoleRepository manages user ↔ role bindings.
type UserRoleRepository interface {
	ListRolesForUser(ctx context.Context, userID uint) ([]Role, error)
	AssignRole(ctx context.Context, userID, roleID uint) error
	RemoveRole(ctx context.Context, userID, roleID uint) error
}

// UserPermissionRepository manages user ↔ permission bindings.
type UserPermissionRepository interface {
	ListPermissionsForUser(ctx context.Context, userID uint) ([]Permission, error)
	PermissionCodesForUser(ctx context.Context, userID uint) (map[string]struct{}, error)
	AssignPermission(ctx context.Context, userID uint, permissionID string) error
	AssignPermissionByName(ctx context.Context, userID uint, permissionName string) error
	RemovePermission(ctx context.Context, userID uint, permissionID string) error
}
