# Session: Role modify permissions (P38–P40)

**Date:** 2026-05-21

## Permission catalog

| ID | `permission_name` | Granted to roles |
|----|-------------------|------------------|
| P38 | `sysadmin:modify` | sysadmin |
| P39 | `admin:modify` | sysadmin, admin |
| P40 | `instructor:modify` | sysadmin, admin, instructor |

**Out of scope:** `middleware.RequirePermission` on HTTP routes (catalog + matrix only).

## Files touched

- `internal/shared/constants/permissions.go` — P38–P40
- `internal/system/application/roles_permission.go` — 6 role bindings
- `migrations/000010_role_modify_permissions.{up,down}.sql`
- `migrations/README.md`, `docs/database.md`, `docs/modules/rbac.md`

## GitNexus impact (pre-edit)

- `SyncPermissions` / `SyncRolePermissions`: **LOW** (sync jobs + system API)
- `AllPermissionEntries`: **HIGH** (additive catalog fields; expected)

## Post-deploy

```bash
cd be-mycourse
# MIGRATE=1 go run .   # or usual migrate
go run ./cmd/syncpermissions
go run ./cmd/syncrolepermissions
```

Users with cached JWTs need **re-login** for new permission codes in claims.

## Manual check

1. Migrate → P38–P40 in `permissions`
2. Sync CLIs → `role_permissions` matches `roles_permission.go`
3. Login as sysadmin / admin / instructor → `/api/auth/me/permissions` shows expected subset
4. Learner must not receive any of the three
