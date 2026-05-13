# Backend Architecture

## Overview

The **MyCourse** backend is a **Go 1.25** monolith (`module mycourse-io-be`) organized with **Domain-Driven Design (DDD)**. Each business capability is a bounded context under `internal/<domain>/`, divided into four layers with a strict dependency rule.

**Tech stack:** Gin (HTTP), GORM + PostgreSQL (persistence), Redis (cache), golang-jwt (auth), Brevo SMTP (email), Backblaze B2 + BunnyCDN (storage/streaming), Uber Zap (logging), Viper/YAML (config).

---

## DDD Layer Model

Each bounded context under `internal/` follows this layer structure:

```
domain/         ← Core business entities, repository interfaces, domain errors
                   No framework dependencies. Pure Go structs and interfaces.

infra/          ← Concrete implementations: GORM repositories, cloud SDK clients,
                   crypto, external API adapters. Implements domain interfaces.

application/    ← Use-case services (e.g. AuthService, MediaService).
                   Orchestrates domain + infra; owns business rules and workflows.

delivery/       ← HTTP handlers, route registration, request/response DTOs,
                   mapping from domain types to API contracts.

jobs/           ← (Optional) Background workers, schedulers, tickers,
                   orphan/cleanup enqueuers. Started from main.go, not from an
                   HTTP request. A module only adds this folder when it owns
                   long-running or scheduled work. See internal/media/jobs/
                   and internal/system/jobs/ for canonical examples.
```

### Dependency rule

```
delivery, jobs → application → infra → domain
```

- `domain` never imports any other layer.
- `application` never imports `delivery` or `jobs`.
- `infra` never imports `application`, `delivery`, or `jobs`.
- `jobs` (when present) may import its own module's `application` and/or `domain` package, but never `delivery`. Treat `jobs` as a peer "entry point" to `delivery`: HTTP-triggered work lives in `delivery`, time-/ticker-triggered work lives in `jobs`.
- Cross-domain dependencies are handled via **interfaces** defined in the consuming domain and adapted in `internal/server/wire.go`.

---

## Bounded Contexts

| Context | Path | Responsibility |
|---------|------|---------------|
| **auth** | `internal/auth/` | User registration, login, email confirmation, JWT sessions, token refresh |
| **media** | `internal/media/` | File/video upload, B2/Bunny storage, orphan cleanup, webhooks |
| **rbac** | `internal/rbac/` | Roles, permissions, user-role/user-permission bindings |
| **taxonomy** | `internal/taxonomy/` | Categories, tags, course levels |
| **system** | `internal/system/` | Privileged operations, RBAC sync, scheduler control |

---

## Shared Infrastructure (`internal/shared/`)

Cross-cutting concerns that are not domain-specific:

| Package | Purpose |
|---------|---------|
| `internal/shared/db/` | GORM setup, PostgreSQL connection, SQL migrations |
| `internal/shared/cache/` | Redis client (`go-redis v9`) |
| `internal/shared/setting/` | YAML config loading with env-var substitution |
| `internal/shared/logger/` | Uber Zap bootstrap, `WithRequestID`, `FromContext` |
| `internal/shared/token/` | JWT generation and validation |
| `internal/shared/middleware/` | Gin middleware: CORS, auth JWT, RBAC permission checks, rate limiting, request logger |
| `internal/shared/response/` | Unified `{ code, message, data }` response envelope |
| `internal/shared/validate/` | Request validation helpers |
| `internal/shared/brevo/` | Brevo SMTP email client |
| `internal/shared/mailtmpl/` | HTML email templates |
| `internal/shared/errors/` | Shared `ErrXXX` sentinel vars and error codes |
| `internal/shared/constants/` | Cross-domain constants (only 5 files: dbschema names, error messages, media limits, permission IDs, register HTTP headers) |
| `internal/shared/utils/` | Generic utilities (image encode, random, fingerprint) |

---

## Composition Root

Dependency injection lives in **`internal/server/wire.go`**. The `Wire()` function:

1. Instantiates all infrastructure (GORM repos, cloud clients, Redis).
2. Constructs application services in dependency order:
   - RBAC (no cross-domain deps)
   - System (no cross-domain deps)
   - Media (no cross-domain deps)
   - Taxonomy (depends on Media for image validation)
   - Auth (depends on RBAC + Media)
3. Wraps cross-domain interface adapters (e.g. `rbacPermissionReader`, `mediaProfileImageValidator`).
4. Returns `*Services` and `*Handlers` structs.

`main.go` calls `server.Wire(db, redis)` and passes the results to `server.InitRouter()`.

---

## HTTP Request Path

```
main.go
  └── setting.Setup()         — load YAML + env vars
  └── logger.InitFromSettings() — init Uber Zap global logger
  └── shareddb.Setup()        — connect PostgreSQL (GORM)
  └── supabasepkg.Setup()     — Supabase HTTP client (optional)
  └── cache.SetupRedis()      — connect Redis
  └── mediainfra.NewCloudClientsFromSetting() — init B2/Bunny SDK
  └── maybeMigrateFromEnv()   — apply SQL migrations if MIGRATE=1
  └── server.Wire(db, redis)  — dependency injection
  └── mediajobs.StartMediaPendingCleanupJob() — background worker
  └── server.InitRouter(svcs, handlers)
        └── gin.New()
        └── middleware.RequestLogger()  — structured access log + X-Request-ID
        └── httperr.Middleware()        — centralized error handling
        └── httperr.Recovery()          — panic recovery
        └── cors.New(...)               — CORS
        └── gzip.Gzip(...)             — response compression
        └── /api/system  ← privileged ops (system JWT, rate limit by IP)
        └── /api/v1 (no-filter) ← webhook callbacks (no auth)
        └── /api/v1 (standard) ← public + authenticated routes
        └── /api/internal-v1  ← RBAC admin (internal API key)
  └── router.Run(":"+port)
```

---

## Route Groups

| Group | Middleware | Purpose |
|-------|-----------|---------|
| `/api/system` | `BeforeInterceptor`, `RateLimitSystemIP(10,3)`, `RequireSystemAccessToken` | Privileged operators: RBAC sync, scheduler control |
| `/api/v1` (no-filter) | none (mounts before `BeforeInterceptor`) | Webhook callbacks that bypass JWT |
| `/api/v1` unauthenticated | `BeforeInterceptor`, `RateLimitLocal(60,1)` | Register, login, confirm, refresh |
| `/api/v1` authenticated | `BeforeInterceptor`, `RateLimitLocal(120,1)`, `AuthJWT` | Protected user endpoints |
| `/api/internal-v1` | `RateLimitLocal(60,1)`, `BeforeInterceptor`, `RequireInternalAPIKey` | Internal RBAC administration |

---

## Configuration

- YAML files under `config/` (`app.yaml`, `app-<STAGE>.yaml`) with values replaced from environment variables.
- `STAGE` environment variable selects the config file (e.g. `STAGE=prod` loads `app-prod.yaml`).
- Key environment variables: `SUPABASE_DB_URL`, `SUPABASE_URL`, `SUPABASE_SERVICE_ROLE_KEY`, `APP_BASE_URL`, `APP_CLIENT_BASE_URL`, `CORS_ALLOWED_ORIGINS`, `MIGRATE`, `CLI_REGISTER_NEW_SYSTEM_USER`.

---

## Background Jobs

| Job | Location | Trigger |
|-----|----------|---------|
| Media pending cleanup worker | `internal/media/jobs/` | Started in `main.go` after wiring |
| RBAC permission sync | `internal/system/` | `/api/system/permission-sync-now` or scheduled |
| Role-permission sync | `internal/system/` | `/api/system/role-permission-sync-now` or scheduled |

---

## GitNexus

```bash
npx gitnexus analyze --force
```

MCP resource `gitnexus://repo/be-mycourse/context` lists current graph stats and staleness. Use `gitnexus_query`, `gitnexus_context`, and `gitnexus_impact` before modifying any symbol.

---

## Related Documentation

| Document | Contents |
|----------|----------|
| [`README.md`](../README.md) | Quick start, CORS, response format, error codes |
| [`docs/folder-structure.md`](folder-structure.md) | Complete directory tree |
| [`docs/router.md`](router.md) | Full API route table |
| [`docs/data-flow.md`](data-flow.md) | End-to-end request flows |
| [`docs/patterns.md`](patterns.md) | Coding conventions and patterns |
| [`docs/modules/auth.md`](modules/auth.md) | Auth domain deep-dive |
| [`docs/modules/media.md`](modules/media.md) | Media domain deep-dive |
| [`docs/modules/rbac.md`](modules/rbac.md) | RBAC domain deep-dive |
| [`docs/modules/taxonomy.md`](modules/taxonomy.md) | Taxonomy domain deep-dive |
