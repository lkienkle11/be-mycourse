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
- `GET /me`, `PATCH /me`, `DELETE /me` (soft), `DELETE /me/hard`
- `GET /me/permissions`
- User access guards via `application/service_access.go` (`checkUserAccessible`: deleted / disabled / `banned_until`)
- Stateful JWT sessions backed by PostgreSQL JSONB
- Redis cache for `/me` responses and login negative cache
- Confirmation email rate limiting (Redis sliding window + Postgres lifetime cap)

**Key files:**
- `domain/` — pure `User` + session map types, repository interfaces, token TTL constants
- `application/AuthService` — orchestrates register/login/confirm/refresh flows
- `infra/` — `userRow`, `gormjsonb`, `user_repo` (user + refresh session), bcrypt — **no `entity` package**
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
- `domain/` — `File`, `MediaGateway` port, repository interfaces, Bunny/webhook/meta types
- `application/MediaService` — upload/delete/list/batch (uses `MediaGateway`, not `infra`)
- `infra/storage_gateway.go` — implements `MediaGateway`; repos + cloud clients
- `delivery/Handler` — Gin handlers + injected `MediaGateway` for multipart/metadata
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
- **Course topics**, **outcomes**, **skills**, **levels**, **tags** — CRUD with soft delete (`DELETE /:id`), hard delete (`DELETE /:id/hard`), list including deleted (`GET /full`)
- Partial unique slug indexes among active rows only (re-create slug after soft delete)
- Optional topic/outcome images linked to `media_files`; orphan cleanup on hard delete only

**Exposed under:** `/api/v1/taxonomy/...` (requires JWT + permission). Convention: **`docs/patterns.md`** (CRUD soft delete).

See [`docs/modules/taxonomy.md`](modules/taxonomy.md) for full deep-dive.

---

### Instructor (`internal/instructor/`)

Instructor roster, applications (approve/reject with reason), profiles, expertise junctions, and support tickets.

**Capabilities:**
- Roster: list/add-by-email/delete (assign/remove **`instructor`** role only — **learner** retained)
- Applications: submit → pending; approve → add instructor role; reject → reason required, no role change
- Profiles: admin CRUD + `/me`; media file validation for CV / intro video
- Expertise: per-user topic/skill links to taxonomy tables
- Tickets + messages; close with P58; no messages when closed
- Stubs: `/instructor-stubs/assignments`, `/activity-log` → coming soon

**Exposed under:** `/api/v1/instructors`, `/instructor-applications`, `/instructor-profiles`, `/instructor-tickets`, … (JWT + permission). Wired in `internal/server/router.go`.

**Migration:** `000013_instructor_management` (not `000011` — that migration is audit timestamps BIGINT).

See [`docs/modules/instructor.md`](modules/instructor.md) for full deep-dive.

---

### System (`internal/system/`)

Privileged operations for system administrators.

**Capabilities:**
- System login (issues short-lived system JWT)
- Permission sync (immediate or scheduled — reads `constants.AllPermissions`)
- Role-permission sync (immediate or scheduled — reads `constants.RolePermissions`)
- Scheduler control: create/stop permission sync job and role-permission sync job

**Exposed under:** `/api/system/...` (requires system access token from CLI login)

**Key files:**
- `domain/ports.go` — `SystemCrypto` port (HMAC credentials + system JWT mint/parse)
- `application/SystemService` — uses `SystemCrypto`; does not import `infra`
- `infra/crypto.go` + `infra/crypto_ports.go` — adapter wired in `server/wire.go`
- `jobs/sync_schedulers.go` — RBAC catalog sync tickers

**Crypto semantics:** `system_app_config.app_cli_system_password`, `app_system_env`, and `app_token_env` are **bcrypt hashes (cost 14)** at rest. CLI registration verifies the app password via `auth/infra.CheckPassword`. Login/register use the hash strings as HMAC/JWT key material through `SystemCrypto` → `shared/cryptox`.

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
