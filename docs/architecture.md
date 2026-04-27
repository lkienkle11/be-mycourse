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

Typical graph stats (refresh after large changes; run `npx gitnexus analyze --force` then read MCP context): on the order of **~767** nodes, **~1,837** edges, **~19** clusters, **~62** execution flows. MCP resource `gitnexus://repo/be-mycourse/context` lists current counts and staleness.

**Functional clusters** (high cohesion areas in the graph) include, among others: **Services** (business logic), **V1** (HTTP handlers under `api/v1`), **Middleware**, **Httperr** / **Response** (cross-cutting HTTP), **Dto**, **Token**, **Setting**, **Constants**, **Dbschema**.

Useful queries (CLI examples; set `-r be-mycourse` when multiple repos are indexed):

- `npx gitnexus query -r be-mycourse "JWT auth refresh"`
- `npx gitnexus context -r be-mycourse InitRouter`

---

## HTTP request path

1. **`main.go`** loads settings, DB, optional privileged-user **CLI** when `CLI_REGISTER_NEW_SYSTEM_USER` is truthy (then exits), Supabase clients, Redis, optional migrate (`MIGRATE=1`), system config bootstrap, queue consumers, then **`api.InitRouter()`**.
2. **`api/router.go`** attaches global middleware: `pkg/httperr` (validation + recovery), **CORS**, **gzip**, then groups under **`/api`**.
3. **`/api/system`** — `BeforeInterceptor`, **`RateLimitSystemIP(10, 3)`** (overridable per IP via `middleware.SetSystemRateLimitOverride`), short-lived system JWT for privileged operators: login + permission / role-permission sync and in-memory 12h jobs (`api/system`, `services/system.go`, `internal/jobs/*`).
4. **`/api/v1`** uses `middleware.BeforeInterceptor()` on all routes, then splits into:
   - **Authenticated subtree** — `RateLimitLocal` + **`middleware.AuthJWT()`** → `api/v1.RegisterAuthenRoutes`.
   - **Unauthenticated subtree** — `RateLimitLocal` only → `api/v1.RegisterNotAuthenRoutes`.
5. **`/api/internal-v1`** — `RateLimitLocal`, `BeforeInterceptor`, **`middleware.RequireInternalAPIKey()`** → internal RBAC HTTP API (`api/v1.RegisterInternalRoutes`).
6. **Handlers** in `api/v1/*.go` parse/bind DTOs, call **`services/*`**, and respond with **`pkg/response`** helpers (never ad-hoc `gin.H` envelopes).

**Redis:** Auth flows and `GET /api/v1/me` use `services/cache` (TTL-backed JSON and negative login cache — see `docs/modules/auth.md`). If Redis is unavailable, behaviour degrades to database-only where helpers no-op on errors.

---

## Directory map (authoritative)

| Path | Role |
|------|------|
| `main.go` | Process entry: settings, DB, cache, migrate flag, bootstrap, queues, router, listen on `setting.ServerSetting.Port` (default **8080**). |
| `api/router.go` | Gin engine, global middleware, `/api/system`, `/api/v1`, `/api/internal-v1` groups. |
| `api/system/` | Privileged system routes (rate limit, system JWT, RBAC sync / job control). |
| `api/v1/` | Versioned handlers and route modules: `auth.go`, `me.go`, `routes.go`, `taxonomy/*`, `internal/*`, … |
| `middleware/` | JWT auth, RBAC permission checks, API key for internal routes, rate limit, shared `BeforeInterceptor`. |
| `services/` | Business logic (`auth.go`, `rbac.go`, …) plus `services/cache/` for Redis. |
| `internal/jobs/` | In-memory 12h RBAC sync tickers started/stopped via `/api/system` (not env-gated). |
| `internal/rbacsync/` | RBAC sync: permissions from `constants.AllPermissions`, role matrix from `constants.RolePermissions`. |
| `dto/` | Request/response and query DTOs; **`dto.BaseFilter`** for list endpoints (see README). |
| `models/` | GORM models and DB setup (`setup.go`, taxonomy models, …). |
| `migrations/` | Versioned SQL migrations (embedded / migrate tooling). |
| `pkg/response` | Unified `{ code, message, data }` (and health shape). |
| `pkg/errcode` | Application error codes. |
| `pkg/httperr` | Gin middleware for errors and panic recovery. |
| `pkg/setting` | YAML config with per-stage files and `.env` substitution. |
| `pkg/token`, `pkg/validate`, `pkg/logger`, `pkg/supabase`, `pkg/envbool`, … | Cross-cutting utilities. |
| `config/` | System bootstrap (`InitSystem`, default configs). |
| `pkg/cache_clients/` | Redis client wiring. |
| `queues/` | Async consumer placeholder. |
| `constants/` | RBAC (`roles.go`, `permissions.go`, `roles_permission.go`), domain enums (e.g. `media.go`), and **`error_msg.go`** — canonical **string literals for errors/sentinels** (and tightly coupled numeric caps such as `MaxMediaUploadFileBytes`). **`pkg/errcode/messages.go`** holds the code→message map but **must import** string constants from here when the same wording is used for sentinels (e.g. **`MsgFileTooLargeUpload`** for `FileTooLarge` + `ErrFileExceedsMaxUploadSize`) so copy cannot drift. |
| `dbschema/` | RBAC-related schema helpers. |
| `cmd/syncpermissions/` | Upsert `permissions.permission_name` by `permission_id` (`//go:generate` from `main.go`). |
| `cmd/syncrolepermissions/` | Rebuild `role_permissions` from `constants/roles_permission.go`. |
| `tracing/`, `runtime/` | Observability / runtime placeholders. |
| `tests/` | **Module-level and integration tests** — Go test packages, shared harnesses, and fixtures that should not live next to production code. Prefer this tree when adding cross-package or black-box suites; small unit tests may still use colocated `*_test.go`. See `.full-project/patterns.md` and `README.md` (**Testing**). |

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
| OPTIONS | `/api/v1/media/files` | JWT + permission middleware |
| GET | `/api/v1/media/files` | JWT + permission middleware |
| POST | `/api/v1/media/files` | JWT + permission middleware |
| OPTIONS | `/api/v1/media/files/:id` | JWT + permission middleware |
| GET | `/api/v1/media/files/:id` | JWT + permission middleware |
| PUT | `/api/v1/media/files/:id` | JWT + permission middleware |
| DELETE | `/api/v1/media/files/:id` | JWT + permission middleware |
| OPTIONS | `/api/v1/media/files/local/:token` | JWT + permission middleware |
| GET | `/api/v1/media/files/local/:token` | JWT + permission middleware |

Exact permission constants for `/me/permissions` are wired in `api/v1/routes.go`.

---

## Internal API (`/api/internal-v1`)

RBAC administration (permissions, roles, user-role and user-direct-permission assignments) is exposed under **`/api/internal-v1/rbac/...`** and protected by **`RequireInternalAPIKey`**. See `api/v1/internal/rbac_handler.go`, `api/v1/internal/routes.go`, and `api/v1/routes.go` for the full route table.

---

## Configuration & environment

- **YAML** under `config/` (`app.yaml`, `app-<STAGE>.yaml`) with values overridden from **environment variables** (see `pkg/setting` and `.env.example`).
- **CORS** allowed origins from `CORS_ALLOWED_ORIGINS` (comma-separated); see README for header contract with the frontend.
- **RBAC sync from constants** — immediate runs or 12h in-memory jobs via **`/api/system`** (`create-*-sync-job`, `delete-*-sync-job`, `*-sync-now`). Secrets live in **`system_app_config`** (singleton row); privileged operators in **`system_privileged_users`**.
- **`CLI_REGISTER_NEW_SYSTEM_USER`** — when truthy, after DB init the binary prompts for CLI app password + new privileged credentials, writes `system_privileged_users`, prints English success/failure, then exits.

---

## Related documentation

| Document | Contents |
|----------|----------|
| [`README.md`](../README.md) | Quick start, CORS, response format, error codes, project structure summary. |
| [`docs/deploy.md`](deploy.md) | Full-stack VPS deploy, Nginx, PM2, CI/CD. |
| [`docs/database.md`](database.md) | Database schema, tables, and migration history. |
| [`docs/requirements.md`](requirements.md) | Functional & non-functional requirements for all features. |
| [`docs/sequence_diagrams.md`](sequence_diagrams.md) | Mermaid sequence diagrams for every system flow. |
| [`docs/return_types.md`](return_types.md) | Go service return types and full JSON response shapes per API. |
| [`docs/curl_api.md`](curl_api.md) | Complete API reference with cURL examples and Postman scripts. |
| [`docs/modules/auth.md`](modules/auth.md) | Auth service and cache behaviour. |
| [`docs/modules/user.md`](modules/user.md) | User profile endpoints — `GET /me`, `GET /me/permissions`. |
| [`docs/modules/media.md`](modules/media.md) | Implemented media upload module (cloud gateway, helper/util split, config-driven provider, **2 GiB** per-file cap + Gin multipart memory tuning). |
| [`docs/modules/course.md`](modules/course.md), [`lesson.md`](modules/lesson.md), [`enrollment.md`](modules/enrollment.md) | Domain module notes (planned features). |

When this repository sits next to the Next.js app in a monorepo (e.g. **`mycourse-full`**), the frontend deploy runbook is at [`../fe-mycourse/docs/deploy.md`](../fe-mycourse/docs/deploy.md).
