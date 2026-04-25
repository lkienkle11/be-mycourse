# be-mycourse Architecture Snapshot

## System Style
- Layered Go monolith with clear transport -> service -> data boundaries.
- Main composition root is `main.go`, which initializes settings, database, cache, migrations, services, queue placeholder, and HTTP router.

## Runtime Layers
- Transport: `api/`, `api/v1/`, `api/system/`.
- Middleware: `middleware/` for JWT auth, API key auth, system token auth, RBAC, rate limiters.
- Business logic: `services/` and `services/cache/`.
- Data layer: `models/`, raw SQL helpers in `services/rbac.go`, and naming helpers in `dbschema/`.
- Cross-cutting packages: `pkg/*` (response envelope, error codes, validation helpers, config, tokens, migration glue).

## Security Model
- JWT access/refresh flow is implemented in `services/auth.go` and `pkg/token/`.
- RBAC is constants-driven (`constants/permissions.go`, `constants/roles_permission.go`) and synchronized to DB via `internal/rbacsync/` plus `cmd/sync*`.
- Permission enforcement is middleware-based (`middleware.RequirePermission`) with JWT context fast-path and DB fallback.

## Database & Migration Architecture
- SQL migrations are embedded from `migrations/*.sql` via `migrations/embed.go`.
- Current schema focuses on users, RBAC, and system tables (`000001_schema.up.sql`).
- Runtime migration path is gated by `MIGRATE=1` in `main.go`.

## Integration/Operational Boundaries
- Redis cache is optional and wrapped by `pkg/cache_clients/redis.go` and `services/cache/auth_user.go`.
- Email side-effect uses Brevo package (`pkg/brevo`).
- Queue subsystem is currently a placeholder (`queues/queues.go`).

## Expansion Points For E-learning CRUD
- Routes: `api/v1/routes.go` and new domain handlers under `api/v1/`.
- DTOs: `dto/`.
- Services: `services/`.
- Models and SQL migrations: `models/`, `migrations/`.
- Authorization extension: `constants/permissions.go` + `constants/roles_permission.go` + sync commands.
