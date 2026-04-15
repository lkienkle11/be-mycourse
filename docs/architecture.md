# Backend Architecture

## Overview

The **MyCourse** backend is a **Go 1.25** monolith (`module mycourse-io-be`): **Gin** for HTTP, **GORM** (and **sqlx** where needed) for PostgreSQL, **Redis** for auth-related caching, optional **Supabase** HTTP + DB helpers, **golang-migrate** SQL files under `migrations/`, and a **unified JSON envelope** via `pkg/response` plus numeric codes in `pkg/errcode`.

The layout follows a **practical layered style** (handlers → services → models/adapters), not a separate `delivery/http` / `core/ports` tree. The composition root is **`main.go`** at the repository root.

---

## GitNexus snapshot

Indexing the repo with GitNexus (run from repo root):

```bash
npx gitnexus analyze --force
```

Typical graph stats (refresh after large changes): on the order of **~84** source files, **~691** symbols, **~1,615** relationships, **~17** clusters, **~55** execution flows. MCP resource `gitnexus://repo/be-mycourse/context` lists current counts and staleness.

**Functional clusters** (high cohesion areas in the graph) include, among others: **Services** (business logic), **V1** (HTTP handlers under `api/v1`), **Middleware**, **Httperr** / **Response** (cross-cutting HTTP), **Dto**, **Token**, **Setting**, **Constants**, **Dbschema**.

Useful queries (CLI examples; set `-r be-mycourse` when multiple repos are indexed):

- `npx gitnexus query -r be-mycourse "JWT auth refresh"`
- `npx gitnexus context -r be-mycourse InitRouter`

---

## HTTP request path

1. **`main.go`** loads settings, DB, Supabase clients, Redis, optional migrate (`MIGRATE=1`), system config bootstrap, optional permission auto-sync job (`AUTO_SYNC_PERMISSION_JOB=true|1|yes|y|on`: run once at startup, then every 12h), queue consumers, then **`api.InitRouter()`**.
2. **`api/router.go`** attaches global middleware: `pkg/httperr` (validation + recovery), **CORS**, **gzip**, then groups under **`/api`**.
3. **`/api/v1`** uses `middleware.BeforeInterceptor()` on all routes, then splits into:
   - **Authenticated subtree** — `RateLimitLocal` + **`middleware.AuthJWT()`** → `api/v1.RegisterAuthenRoutes`.
   - **Unauthenticated subtree** — `RateLimitLocal` only → `api/v1.RegisterNotAuthenRoutes`.
4. **`/api/internal-v1`** — `RateLimitLocal`, `BeforeInterceptor`, **`middleware.RequireInternalAPIKey()`** → internal RBAC HTTP API (`api/v1.RegisterInternalRoutes`).
5. **Handlers** in `api/v1/*.go` parse/bind DTOs, call **`services/*`**, and respond with **`pkg/response`** helpers (never ad-hoc `gin.H` envelopes).

**Redis:** Auth flows and `GET /api/v1/me` use `services/cache` (TTL-backed JSON and negative login cache — see `docs/modules/auth.md`). If Redis is unavailable, behaviour degrades to database-only where helpers no-op on errors.

---

## Directory map (authoritative)

| Path | Role |
|------|------|
| `main.go` | Process entry: settings, DB, cache, migrate flag, bootstrap, queues, router, listen on `setting.ServerSetting.Port` (default **8080**). |
| `api/router.go` | Gin engine, global middleware, `/api/v1` and `/api/internal-v1` groups. |
| `api/v1/` | Versioned handlers: `auth.go`, `me.go`, `routes.go`, `internal_rbac.go`, … |
| `middleware/` | JWT auth, RBAC permission checks, API key for internal routes, rate limit, shared `BeforeInterceptor`. |
| `services/` | Business logic (`auth.go`, `rbac.go`, …) plus `services/cache/` for Redis. |
| `internal/jobs/` | App-private background schedulers (e.g. permission auto-sync ticker). |
| `internal/rbacsync/` | App-private RBAC permission catalog sync core (`constants` → DB). |
| `dto/` | Request/response and query DTOs; **`dto.BaseFilter`** for list endpoints (see README). |
| `models/` | GORM models and DB setup (`setup.go`, `repository.go`, …). |
| `migrations/` | Versioned SQL migrations (embedded / migrate tooling). |
| `pkg/response` | Unified `{ code, message, data }` (and health shape). |
| `pkg/errcode` | Application error codes. |
| `pkg/httperr` | Gin middleware for errors and panic recovery. |
| `pkg/setting` | YAML config with per-stage files and `.env` substitution. |
| `pkg/token`, `pkg/validate`, `pkg/logger`, `pkg/supabase`, … | Cross-cutting utilities. |
| `config/` | System bootstrap (`InitSystem`, default configs). |
| `cache_clients/` | Redis client wiring. |
| `queues/` | Async consumer placeholder. |
| `constants/` | Role and permission codes for RBAC. |
| `dbschema/` | RBAC-related schema helpers. |
| `cmd/syncpermissions/` | Permission sync (`//go:generate` from `main.go`). |
| `utils/` | Shared helpers reused across modules (e.g. env boolean parsing). |
| `tracing/`, `runtime/` | Observability / runtime placeholders. |

---

## Public API surface (`/api/v1`)

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/health` | No |
| POST | `/api/v1/auth/register` | No |
| POST | `/api/v1/auth/login` | No |
| GET | `/api/v1/auth/confirm` | No |
| POST | `/api/v1/auth/refresh` | No |
| GET | `/api/v1/me` | JWT |
| GET | `/api/v1/me/permissions` | JWT + permission middleware |

Exact permission constants for `/me/permissions` are wired in `api/v1/routes.go`.

---

## Internal API (`/api/internal-v1`)

RBAC administration (permissions, roles, user-role and user-direct-permission assignments) is exposed under **`/api/internal-v1/rbac/...`** and protected by **`RequireInternalAPIKey`**. See `api/v1/internal_rbac.go` and `api/v1/routes.go` for the full route table.

---

## Configuration & environment

- **YAML** under `config/` (`app.yaml`, `app-<STAGE>.yaml`) with values overridden from **environment variables** (see `pkg/setting` and `.env.example`).
- **CORS** allowed origins from `CORS_ALLOWED_ORIGINS` (comma-separated); see README for header contract with the frontend.
- **Permission auto-sync** from constants to DB can run in background when `AUTO_SYNC_PERMISSION_JOB` is one of `true`, `1`, `yes`, `y`, `on` (runs once at startup, then every 12 hours; implemented in `internal/jobs/permission_sync_scheduler.go`).

---

## Related documentation

| Document | Contents |
|----------|----------|
| [`README.md`](../README.md) | Quick start, CORS, response format, error codes, project structure summary. |
| [`docs/deploy.md`](deploy.md) | Full-stack VPS deploy, Nginx, PM2, CI/CD. |
| [`docs/database.md`](database.md) | Database notes. |
| [`docs/modules/auth.md`](modules/auth.md) | Auth service and cache behaviour. |
| [`docs/modules/user.md`](modules/user.md), [`course.md`](modules/course.md), [`lesson.md`](modules/lesson.md), [`enrollment.md`](modules/enrollment.md) | Domain module notes (extend as features land). |

When this repository sits next to the Next.js app in a monorepo (e.g. **`mycourse-full`**), the frontend deploy runbook is at [`../fe-mycourse/docs/deploy.md`](../fe-mycourse/docs/deploy.md).
