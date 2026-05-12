# MyCourse Backend

## Documentation Convention (Mandatory)

The `docs/` folder is the **primary and authoritative documentation source** for this project.

- **Before starting any task** (coding, planning, debugging, refactoring), read the relevant files in `docs/` first.
- If `docs/` already contains sufficient and up-to-date information → **reuse it directly** without re-running full discovery.
- If `docs/` is missing information or outdated → re-run discovery and **update `docs/` before proceeding**.
- Always sync `docs/` after completing any task that changes architecture, APIs, data flow, patterns, or reusable assets.

| Doc | Contents |
|-----|----------|
| [`docs/architecture.md`](docs/architecture.md) | DDD layering, bounded contexts, dependency direction, HTTP request path |
| [`docs/folder-structure.md`](docs/folder-structure.md) | Full directory tree with purpose of every folder and subfolder |
| [`docs/data-flow.md`](docs/data-flow.md) | How data moves through the system end-to-end |
| [`docs/router.md`](docs/router.md) | Route groups, all API routes, middleware chains |
| [`docs/patterns.md`](docs/patterns.md) | Coding patterns, conventions, error handling, constants, testing layout |
| [`docs/dependencies.md`](docs/dependencies.md) | Key libraries, frameworks, and their relationships |
| [`docs/modules.md`](docs/modules.md) | Module responsibilities, ownership boundaries, testing layout |
| [`docs/modules/auth.md`](docs/modules/auth.md) | Auth module deep-dive (JWT sessions, Redis cache, endpoints) |
| [`docs/modules/media.md`](docs/modules/media.md) | Media module deep-dive (B2/Bunny, upload pipeline, webhooks) |
| [`docs/modules/rbac.md`](docs/modules/rbac.md) | RBAC module deep-dive (roles, permissions, sync) |
| [`docs/modules/taxonomy.md`](docs/modules/taxonomy.md) | Taxonomy module deep-dive (categories, tags, course levels) |

---

## Architecture Overview

The **MyCourse** backend is a Go 1.25 monolith organized with **Domain-Driven Design (DDD)**. Each bounded context lives under `internal/<domain>/` with four layers:

```
domain/       ← entities, repository interfaces, domain errors (no framework deps)
infra/        ← GORM repositories, cloud SDK clients, crypto implementations
application/  ← use-case services (AuthService, MediaService, …)
delivery/     ← Gin HTTP handlers, route registration, DTOs, mapping
```

Dependency rule: `delivery` → `application` → `infra` → `domain`. The `domain` layer never imports any other layer.

Bounded contexts: **auth**, **media**, **rbac**, **taxonomy**, **system**.

Shared infrastructure lives in `internal/shared/` (db, cache, logger, middleware, token, response, etc.).

Dependency injection is performed in `internal/server/wire.go` — all `Services` and `Handlers` are constructed there and passed to `internal/server/router.go`.

---

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL
- Redis
- `libvips-dev` and `pkg-config` (for CGO image encoding — `bimg`)

### Setup

1. Copy `.env.example` to `.env` and fill required keys:

```env
# Database
SUPABASE_DB_URL=...           # Primary Postgres connection (pooler or direct)
SUPABASE_URL=...              # Supabase project URL
SUPABASE_SERVICE_ROLE_KEY=... # Supabase service role key

# Server
APP_BASE_URL=https://api.mycourse.io   # Public base URL (no trailing slash)
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://mycourse.io

# Optional: run DB migrations on startup
# MIGRATE=1

# Optional: register first system user then exit
# CLI_REGISTER_NEW_SYSTEM_USER=1

# Logging (all optional)
LOG_LEVEL=info          # debug | info | warn | error
LOG_FORMAT=json         # json | console
LOG_FILE_PATH=          # append-only NDJSON file for Filebeat (empty = disabled)
LOG_SERVICE_NAME=be-mycourse
LOG_ENVIRONMENT=development
APP_VERSION=0.1.0
```

2. Install dependencies and run:

```bash
go mod tidy
go run .
```

3. Verify:

```bash
curl http://localhost:8080/api/v1/health
```

### Database migrations

Set `MIGRATE=1` in the environment before starting to apply pending SQL migrations:

```bash
MIGRATE=1 go run .
```

### RBAC sync

After first deploy (or whenever permissions/role-permission bindings change):

```bash
# Upsert permissions from constants.AllPermissions
go run ./cmd/syncpermissions

# Rebuild role_permissions from constants.RolePermissions
go run ./cmd/syncrolepermissions
```

Alternatively, use the system API after startup:

```
POST /api/system/login
POST /api/system/permission-sync-now
POST /api/system/role-permission-sync-now
```

---

## Testing

Tests are **co-located** with their packages — not in a separate `tests/` directory.

| Test file | Package |
|-----------|---------|
| `internal/media/application/batch_delete_test.go` | `application_test` |
| `internal/media/application/batch_upload_test.go` | `application_test` |
| `internal/media/infra/pipeline_test.go` | `infra_test` |
| `internal/media/infra/orphan_safety_test.go` | `infra_test` |
| `internal/media/infra/orphan_image_test.go` | `infra_test` |
| `internal/media/infra/bunny_webhook_test.go` | `infra_test` |
| `internal/media/delivery/server_owned_test.go` | `delivery_test` |
| `internal/shared/utils/webp_test.go` | `utils_test` |
| `internal/shared/logger/logger_test.go` | `logger_test` |

### Local verification

```bash
gofmt -w .
go vet ./...
go test ./...
golangci-lint run
go build ./...
```

Use `make build` when `CGO_ENABLED=1` and `libvips-dev` are available.

---

## CORS

Configured via the `CORS_ALLOWED_ORIGINS` environment variable — a comma-separated list of allowed origins.

```env
# .env (local / dev)
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# .env.staging / .env.prod
CORS_ALLOWED_ORIGINS=https://mycourse.io,https://www.mycourse.io
```

Allowed methods: `GET POST PUT PATCH DELETE OPTIONS`
Allowed headers: `Origin Content-Type Authorization X-API-Key X-Refresh-Token X-Session-Id`
Exposed headers: `X-Token-Expired X-Mycourse-Register-Retry-After Retry-After`
Credentials: enabled (`AllowCredentials: true`)

| Custom header | Direction | Purpose |
|---|---|---|
| `Authorization` | request | Bearer access token for all protected endpoints |
| `X-Refresh-Token` | request | Refresh JWT sent to `POST /api/v1/auth/refresh` |
| `X-Session-Id` | request | Session ID sent to `POST /api/v1/auth/refresh` |
| `X-Token-Expired` | response | `"true"` when a 401 is caused by an expired access JWT |

---

## API Response Format

All responses are JSON objects with a standard envelope:

```json
{
  "code":    0,
  "message": "ok",
  "data":    <value>
}
```

| Field | Type | Description |
|-------|------|-------------|
| `code` | `number` | `0` = success. Non-zero = error (see error codes) |
| `message` | `string` | Human-readable status or error message |
| `data` | `null \| object \| array \| PaginatedData` | Response payload |

### Health response (`GET /api/v1/health`)

```json
{ "code": 0, "message": "ok", "status": "ok" }
```

### Paginated data

```json
{
  "code": 0, "message": "ok",
  "data": {
    "result": [...],
    "page_info": { "page": 1, "per_page": 20, "total_pages": 5, "total_items": 98 }
  }
}
```

### Error response

```json
{ "code": 4002, "message": "Invalid email or password", "data": null }
```

### Application error codes

| Code | Constant | Meaning |
|------|----------|---------|
| `0` | `Success` | Operation completed successfully |
| `1001` | `InvalidJSON` | Request body is not valid JSON |
| `2001` | `ValidationFailed` | Request validation failed |
| `2002` | `ValidationField` | Per-field validation error |
| `2003` | `FileTooLarge` | Uploaded file exceeds the 2 GiB per-part limit |
| `2004` | `ExecutableUploadRejected` | Executable/script file rejected |
| `2005` | `MediaMultipartTotalTooLarge` | Aggregate multipart body exceeds 2 GiB |
| `2006` | `MediaTooManyFilesInRequest` | More than 5 parts in a single request |
| `3001` | `BadRequest` | Bad request |
| `3002` | `Unauthorized` | Unauthorized |
| `3003` | `Forbidden` | Forbidden |
| `3004` | `NotFound` | Resource not found |
| `3005` | `Conflict` | Conflict (e.g. duplicate resource) |
| `3006` | `TooManyRequests` | Rate limit exceeded |
| `4001` | `EmailAlreadyExists` | Email address already registered |
| `4002` | `InvalidCredentials` | Invalid email or password |
| `4003` | `WeakPassword` | Password does not meet strength requirements |
| `4004` | `EmailNotConfirmed` | Email address not confirmed yet |
| `4005` | `UserDisabled` | Account has been disabled |
| `4006` | `InvalidConfirmToken` | Invalid or expired confirmation token |
| `4007` | `InvalidSession` | Session unknown, missing, or UUID mismatch |
| `4008` | `RefreshTokenExpired` | Session has expired — re-login required |
| `4009` | `RegistrationAbandoned` | Pending registration removed |
| `4010` | `RegistrationEmailRateLimited` | Too many confirmation emails in sliding window |
| `4011` | `ConfirmationEmailSendFailed` | Confirmation email could not be sent |
| `9001` | `InternalError` | Internal server error |
| `9010` | `B2BucketNotConfigured` | B2 storage not configured |
| `9011` | `BunnyStreamNotConfigured` | Bunny Stream not configured |
| `9017` | `ImageEncodeBusy` | WebP encode gate at capacity |
| `9998` | `Panic` | Unhandled panic |
| `9999` | `Unknown` | Unknown error |

---

## Response Helpers (`internal/shared/response`)

Use helpers in `internal/shared/response` to write responses from handlers. Never write raw `gin.H` envelopes.

```go
response.Health(c)
response.OK(c, "ok", data)
response.Created(c, "created", data)
response.OKPaginated(c, "ok", rows, pageInfo)
response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "bad input", nil)
response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "not authenticated", nil)
```

---

## GET List API — Query Filter Params

Every GET list endpoint embeds `BaseFilter` in its DTO for consistent pagination, sorting, and search:

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | `number` | `1` | Page number (1-based) |
| `per_page` | `number` | `20` | Items per page (max 100) |
| `sort_by` | `string` | — | Field name to sort by |
| `sort_order` | `string` | `asc` | `asc` or `desc` |
| `search_by` | `string` | — | Field name to search in |
| `search_data` | `string` | — | Search term |

---

## CI/CD

Pushing to **`master`** triggers `.github/workflows/deploy-dev.yml`: the test job runs **`go test`**, **`go vet`**, **`golangci-lint run`**, and **`make build`** (CGO + libvips on the runner), then the pipeline builds `mycourse-io-be-dev`, `rsync`s to the server, and runs `scripts/pm2-reload-with-binary-rollback.sh`. See [`docs/deploy.md`](docs/deploy.md) for the full runbook.
