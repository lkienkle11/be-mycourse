# Module: RBAC (Role-Based Access Control)

Manages roles, permissions, and role-permission assignments. This is an **internal admin** module — its endpoints are restricted to internal/admin actors and are not exposed to regular authenticated users.

---

## Responsibility

| Concern | Description |
|---------|-------------|
| Role management | Create and query roles (e.g. admin, instructor, student) |
| Permission management | Define granular permissions (e.g. `course:create`, `media:delete`) |
| Role-permission assignment | Bind permissions to roles; query effective permissions for a role |
| Seed | Pre-populate base roles and permissions at startup via `rbac_seed.go` |

---

## Directory Layout

```
api/v1/internal/
├── rbac_handler.go       # HTTP handlers for role/permission management
└── routes.go             # Route registration for /api/v1/internal/*

services/
├── rbac.go               # Core RBAC service: role/permission CRUD, binding logic
└── rbac_seed.go          # Seeding logic: creates base roles and permissions on startup

middleware/
└── rbac.go               # Middleware: RequirePermission(perm string) enforces access control
```

---

## API Endpoints

All endpoints are under `/api/v1/internal/` and require **internal/admin** authorization.

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/internal/roles` | `ListRoles` | List all roles |
| POST | `/api/v1/internal/roles` | `CreateRole` | Create a new role |
| GET | `/api/v1/internal/permissions` | `ListPermissions` | List all permissions |
| POST | `/api/v1/internal/permissions` | `CreatePermission` | Create a new permission |
| POST | `/api/v1/internal/roles/:id/permissions` | `AssignPermission` | Assign permission(s) to a role |
| DELETE | `/api/v1/internal/roles/:id/permissions/:pid` | `RevokePermission` | Remove permission from a role |
| GET | `/api/v1/internal/roles/:id/permissions` | `GetRolePermissions` | List permissions for a role |

---

## Data Flow

```
HTTP Request
  └─ middleware/rbac.go  (RequirePermission → checks token claims)
       └─ api/v1/internal/rbac_handler.go  (bind DTO, validate)
            └─ services/rbac.go  (business logic, DB operations)
                 └─ database (roles, permissions, role_permissions tables)
                      └─ HTTP Response
```

### Startup Seeding

```
Application boot
  └─ services/rbac_seed.go  (SeedRBACData)
       └─ Upsert base roles: admin, instructor, student
       └─ Upsert base permissions (P1–P13 fixed, P14+ extensible)
       └─ Bind permissions to roles (additive only — never removes existing bindings)
```

---

## Middleware: `RequirePermission`

Located in `middleware/rbac.go`. Used throughout all protected endpoints:

```go
router.Use(middleware.RequirePermission("course:create"))
```

- Reads the authenticated user's role from the JWT claims.
- Loads the role's effective permissions from cache (`services/cache/auth_user.go`) or DB.
- Returns `403 Forbidden` if the required permission is not present.
- Returns `401 Unauthorized` if no valid JWT is provided.

---

## Permission Naming Convention

```
<resource>:<action>
```

Examples:

| Permission | Meaning |
|-----------|---------|
| `course:create` | Create a new course |
| `course:update` | Update any course |
| `course:delete` | Delete any course |
| `media:upload` | Upload media files |
| `media:delete` | Delete media files |
| `user:list` | List all users |
| `rbac:manage` | Manage roles and permissions |

---

## Role Hierarchy (Base Seed)

| Role | Description | Base Permissions |
|------|-------------|-----------------|
| `admin` | Full access | All permissions (P1–P13 + all P14+) |
| `instructor` | Content creator | Course CRUD on own content, media upload |
| `student` | Learner | Read-only on courses, enrollment management |

---

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `middleware/auth_jwt` | JWT validation (must run before RBAC middleware) |
| `services/cache/auth_user.go` | Caches role-permission mappings to reduce DB load |
| `pkg/errcode` | Standard error codes (`ErrPermissionDenied`, `ErrUnauthorized`) |
| `pkg/response` | Standard response envelope |

---

## Critical Rules

- **Permission bindings are additive only**: the seeder NEVER removes existing P1–P13 permissions or changes existing roles. New permissions are added as P14+.
- **Role names are stable**: do not rename `admin`, `instructor`, or `student` — JWT claims reference role names directly.
- **Cache invalidation**: after any role-permission change, the auth user cache (`services/cache/auth_user.go`) must be invalidated for affected users. Affected users must re-login to receive updated JWT claims.

---

## Reusable Assets

| Asset | Type | Location | Notes |
|-------|------|----------|-------|
| `RequirePermission` | Middleware | `middleware/rbac.go` | Use on any route requiring access control |
| Role/Permission DTOs | Data type | `dto/` | Request/response shapes for RBAC operations |
| RBAC seed function | Utility | `services/rbac_seed.go` | Call at startup to ensure base data exists |
