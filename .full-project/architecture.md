# be-mycourse Architecture Snapshot


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## System Style
- Layered Go monolith with clear transport -> service -> data boundaries.
- Main composition root is `main.go`, which initializes settings, database, cache, migrations, services, queue placeholder, and HTTP router.

## Runtime Layers
- Transport: `api/`, `api/v1/`, `api/v1/internal/`, `api/system/`.
- Middleware: `middleware/` for JWT auth, API key auth, system token auth, RBAC, rate limiters.
- Business logic: `services/` and `services/cache/`.
- Data layer: `models/`, raw SQL helpers in `services/rbac.go`, and naming helpers in `dbschema/`.
- Cross-cutting packages: `pkg/*` (response envelope, error codes, validation helpers, config, tokens, migration glue, cache client, shared entities).

## Security Model
- JWT access/refresh flow is implemented in `services/auth.go` and `pkg/token/`.
- RBAC is constants-driven (`constants/permissions.go`, `constants/roles_permission.go`) and synchronized to DB via `internal/rbacsync/` plus `cmd/sync*`.
- Permission enforcement is middleware-based (`middleware.RequirePermission`) with JWT context fast-path and DB fallback.
- Route boundary note: `/api/v1` now has a dedicated no-filter registration lane (`RegisterNoFilterRoutes`) for public webhooks that must bypass `BeforeInterceptor`/auth/permission middleware.

## Database & Migration Architecture
- SQL migrations are embedded from `migrations/*.sql` via `migrations/embed.go`.
- Current schema focuses on users, RBAC, and system tables (`000001_schema.up.sql`).
- Runtime migration path is gated by `MIGRATE=1` in `main.go`.

## Testing layout
- **All tests** (unit/module-level/integration) and shared test harnesses belong under repository root **`tests/`** (see `.full-project/patterns.md`).

## Integration/Operational Boundaries
- Redis cache is optional and wrapped by `pkg/cache_clients/redis.go` and `services/cache/auth_user.go`.
- Shared pure domain entities currently live in `pkg/entities/*` and are embedded by taxonomy models.
- Email side-effect uses Brevo package (`pkg/brevo`).
- Queue subsystem is currently a placeholder (`queues/queues.go`).

## Expansion Points For E-learning CRUD
- Routes: `api/v1/routes.go` and new domain handlers under `api/v1/`.
- DTOs: `dto/`.
- Services: `services/`.
- Models and SQL migrations: `models/`, `migrations/`.
- Authorization extension: `constants/permissions.go` + `constants/roles_permission.go` + sync commands.
- Media upload expansion now implemented via:
  - `api/v1/media/*` (transport)
  - `services/media/*` (provider dispatch + upload flow logic + third-party SDK clients)
  - `pkg/media/*` startup-initialized SDK clients (`pkg/media.Setup()` in `main.go`)
  - `pkg/entities/file.go` + `constants/media.go` (shared descriptor + enums)
  - persisted metadata design (`media_files` + `models/media_file.go` + `repository/media/file_repository.go`) with cloud↔DB sync workflow
