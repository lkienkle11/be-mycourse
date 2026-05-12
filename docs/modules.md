# Module Responsibilities

## Architecture Pattern

Every module lives under `internal/<domain>/` and follows the DDD layer structure:

```
domain/       ← entities, interfaces, errors
infra/        ← GORM repos, external SDK clients
application/  ← use-case service
delivery/     ← HTTP handlers, routes, DTOs
jobs/         ← (if any) background workers / schedulers — started from main.go
```

A module only adds `jobs/` when it owns scheduled or long-running background work (e.g. `internal/media/jobs/` for pending cloud cleanup, `internal/system/jobs/` for RBAC sync schedulers). Modules without background work simply omit the folder.

See [`docs/architecture.md`](architecture.md) for the full dependency rule.

---

## Implemented Modules

### Auth (`internal/auth/`)

Handles user lifecycle and session management.

**Capabilities:**
- Register (pending user + confirmation email via Brevo)
- Login / email confirmation / token refresh
- `GET /me`, `PATCH /me`, `DELETE /me`
- `GET /me/permissions`
- Stateful JWT sessions backed by PostgreSQL JSONB
- Redis cache for `/me` responses and login negative cache
- Confirmation email rate limiting (Redis sliding window + Postgres lifetime cap)

**Key files:**
- `domain/` — `User` entity, `UserRepository` and `RefreshSessionRepository` interfaces, token TTL constants
- `application/AuthService` — orchestrates register/login/confirm/refresh flows
- `infra/GormUserRepository`, `infra/GormRefreshSessionRepository` — Postgres persistence
- `delivery/Handler` — Gin handlers; `routes.go` — route registration

See [`docs/modules/auth.md`](modules/auth.md) for full deep-dive.

---

### Media (`internal/media/`)

Unified API for file and video uploads with cloud storage providers.

**Capabilities:**
- Multipart file upload (1–5 parts per request) to Backblaze B2 or Bunny Stream
- Synchronous WebP encoding for all image uploads (bimg/libvips, CGO required)
- Executable/script file denylist
- Video status polling via Bunny Stream API
- Bunny webhook processing (signature verification + metadata sync)
- Orphan cleanup: deferred deletion of superseded cloud objects
- Batch delete (up to 10 keys)
- `GET /media/files/local/:token` — decode reversible signed URL tokens

**Key files:**
- `domain/` — `File` entity, `FileRepository`/`PendingCleanupRepository` interfaces, Bunny status codes, webhook types, metadata key constants
- `application/MediaService` — upload/delete/list/batch orchestration
- `infra/` — GORM repos, B2 and Bunny SDK clients, metadata inference, WebP encoding pipeline
- `delivery/Handler` — Gin handlers; `routes.go`; `RegisterWebhookRoutes`
- `jobs/` — `OrphanEnqueuer`, cleanup scheduler, `GlobalCounters`

See [`docs/modules/media.md`](modules/media.md) for full deep-dive.

---

### RBAC (`internal/rbac/`)

Role-based access control — internal admin API.

**Capabilities:**
- Permission CRUD (create, list, update, delete)
- Role CRUD (create, list, get, update, delete)
- Set role permissions (full replace)
- Assign/remove roles to/from users
- Assign/remove direct permissions to/from users
- Query effective permissions for a user

**Exposed under:** `/api/internal-v1/rbac/...` (requires internal API key)

See [`docs/modules/rbac.md`](modules/rbac.md) for full deep-dive.

---

### Taxonomy (`internal/taxonomy/`)

Reference data for classifying course content.

**Capabilities:**
- **Categories**: hierarchical or flat subject groupings with optional image (linked to `media_files`)
- **Course Levels**: difficulty designations (Beginner, Intermediate, Advanced)
- **Tags**: free-form keyword labels

**Exposed under:** `/api/v1/taxonomy/...` (requires JWT + permission)

See [`docs/modules/taxonomy.md`](modules/taxonomy.md) for full deep-dive.

---

### System (`internal/system/`)

Privileged operations for system administrators.

**Capabilities:**
- System login (issues short-lived system JWT)
- Permission sync (immediate or scheduled — reads `constants.AllPermissions`)
- Role-permission sync (immediate or scheduled — reads `constants.RolePermissions`)
- Scheduler control: create/stop permission sync job and role-permission sync job

**Exposed under:** `/api/system/...` (requires system access token after login)

---

## Planned But Not Implemented

- **Course module** (phase 02+)
- **Lesson module** (phase 05+)
- **Enrollment module** (phase 11+)

These currently have no route/service/infra implementations.

---

## Shared Infrastructure (`internal/shared/`)

Not a business module — provides cross-cutting concerns consumed by all domains:

- `db/` — PostgreSQL connection, GORM setup, migration runner
- `cache/` — Redis client
- `setting/` — YAML + env config
- `logger/` — Uber Zap
- `middleware/` — auth JWT, RBAC permission check, rate limiting, request logger
- `response/` — standard JSON envelope
- `token/` — JWT sign/parse
- `brevo/` — Brevo SMTP client
- `mailtmpl/` — email templates
- `validate/` — request validation
- `utils/` — image encoding, fingerprinting, random
- `errors/` — shared sentinel errors
- `constants/` — cross-domain constants (5 files)

---

## Ownership and Boundaries

- Middleware and RBAC permission enforcement are high-risk shared infrastructure — changes require careful impact analysis.
- Each domain's `application/` service is the single authority for that domain's business rules.
- Cross-domain calls (e.g. Auth calling RBAC for permission codes) are mediated through **interfaces** injected in `internal/server/wire.go` — never direct imports between domain application layers.
- New domain CRUD plugs into existing route/service/infra patterns without changing the RBAC engine.

---

## Testing

Tests are **co-located** with their packages. See [`docs/patterns.md`](patterns.md) for the full testing convention and list of existing test files.

```bash
go test ./...
```
