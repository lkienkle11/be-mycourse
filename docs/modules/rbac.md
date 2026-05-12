# RBAC Module

The RBAC module (`internal/rbac/`) manages roles, permissions, and user-role/user-permission bindings. It is an **internal admin** module — endpoints are restricted to internal actors via an API key and are not exposed to regular users.

---

## Directory Layout

```
internal/rbac/
├── domain/
│   ├── permission.go        # Permission entity
│   └── role.go              # Role entity
├── application/
│   └── rbac_service.go      # RBACService: full permission/role/user binding CRUD
├── infra/
│   ├── gorm_permission_repo.go
│   ├── gorm_role_repo.go
│   ├── gorm_user_role_repo.go
│   ├── gorm_user_permission_repo.go
│   └── sql_templates.go     # Raw SQL for complex joins
└── delivery/
    ├── handler.go            # HTTP handlers
    └── routes.go             # Route registration under /api/internal-v1/rbac
```

---

## API Endpoints

All endpoints are under `/api/internal-v1/rbac/` and require the `X-API-Key` internal API key header.

### Permissions

| Method | Path | Description |
|--------|------|-------------|
| GET | `/rbac/permissions` | List all permissions |
| POST | `/rbac/permissions` | Create a new permission |
| PATCH | `/rbac/permissions/:permissionId` | Update a permission |
| DELETE | `/rbac/permissions/:permissionId` | Delete a permission |

### Roles

| Method | Path | Description |
|--------|------|-------------|
| GET | `/rbac/roles` | List all roles |
| POST | `/rbac/roles` | Create a new role |
| GET | `/rbac/roles/:id` | Get role by ID |
| PATCH | `/rbac/roles/:id` | Update a role |
| PUT | `/rbac/roles/:id/permissions` | Set (replace) all permissions for a role |
| DELETE | `/rbac/roles/:id` | Delete a role |

### User Bindings

| Method | Path | Description |
|--------|------|-------------|
| GET | `/rbac/users/:userId/roles` | List roles assigned to a user |
| GET | `/rbac/users/:userId/permissions` | List effective permissions for a user (roles + direct) |
| GET | `/rbac/users/:userId/direct-permissions` | List only direct-assigned permissions |
| POST | `/rbac/users/:userId/roles` | Assign a role to a user |
| DELETE | `/rbac/users/:userId/roles/:roleId` | Remove a role from a user |
| POST | `/rbac/users/:userId/direct-permissions` | Assign a direct permission to a user |
| DELETE | `/rbac/users/:userId/direct-permissions/:permissionId` | Remove a direct permission from a user |

---

## Permission Middleware

`middleware.RequirePermission(checker, actions...)` is used throughout all protected endpoints:

```go
// Example: require media_file:read permission
media.GET("", middleware.RequirePermission(pc, constants.AllPermissions.MediaFileRead), h.listFiles)
```

- Reads JWT embedded permissions first (fast path).
- If the required permission is absent from claims, falls back to DB lookup via `RBACService.PermissionCodesForUser`.
- Returns `403 Forbidden` if the permission is not present.
- Returns `401 Unauthorized` if no valid JWT is provided.

---

## Permission Naming Convention

```
<resource>:<action>
```

Examples:

| Permission Name | Meaning |
|----------------|---------|
| `user:read` | Read user data |
| `media_file:read` | List/get media files |
| `media_file:create` | Upload media files |
| `media_file:delete` | Delete media files |
| `category:create` | Create a taxonomy category |
| `tag:update` | Update a taxonomy tag |

---

## RBAC Sync

Permission and role-permission bindings are managed as **code** in constants and synced to the database:

```bash
# Upsert permission names from constants.AllPermissions
go run ./cmd/syncpermissions

# Rebuild role_permissions table from constants.RolePermissions
go run ./cmd/syncrolepermissions
```

Or via the system API:
```
POST /api/system/permission-sync-now
POST /api/system/role-permission-sync-now
```

---

## Cross-Domain Usage

Auth and other domains need to check permissions without importing the RBAC application layer. This is handled via the `PermissionChecker` interface defined in `internal/shared/middleware/`:

```go
type PermissionChecker interface {
    PermissionCodesForUser(userID uint) (map[string]struct{}, error)
}
```

`internal/server/wire.go` adapts `RBACService` to this interface and injects it into the middleware and all domain handlers that need permission checking.

---

## Critical Rules

- **Permission bindings are additive** in the seeder: it never removes existing permissions.
- **Role names are stable**: do not rename `admin`, `instructor`, `learner` — JWT claims reference role names.
- **Cache invalidation**: after role-permission changes, affected users must re-login to receive updated JWT claims (no real-time push).

---

## Implementation Reference

| Concern | Location |
|---------|----------|
| RBACService | `internal/rbac/application/rbac_service.go` |
| GORM repositories | `internal/rbac/infra/` |
| HTTP handlers | `internal/rbac/delivery/handler.go` |
| Route registration | `internal/rbac/delivery/routes.go` |
| Permission middleware | `internal/shared/middleware/` |
| Permission constants catalog | `internal/shared/constants/permissions.go` |
| Permission sync CLI | `cmd/syncpermissions/main.go` |
| Role-permission sync CLI | `cmd/syncrolepermissions/main.go` |
| Cross-domain interface adapter | `internal/server/wire.go` (`rbacPermissionReader`, `rbacPermissionUseCase`) |
