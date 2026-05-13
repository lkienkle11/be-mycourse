# Patterns and Conventions

## DDD Layer Conventions

### Layer responsibilities

| Layer | What goes here |
|-------|---------------|
| `domain/` | Entities (plain Go structs), repository interfaces, domain-specific errors, domain constants (TTLs, limits relevant only to that domain) |
| `infra/` | GORM repository implementations, cloud SDK clients, external API adapters, crypto implementations |
| `application/` | Use-case services — orchestrate domain + infra; no HTTP types |
| `delivery/` | Gin handlers, route registration, request/response DTOs, mapping from domain/application types to API shapes |
| `jobs/` *(if any)* | Background workers, schedulers, tickers, and orphan/cleanup enqueuers. Started from `main.go` (not by an HTTP request). Examples: `internal/media/jobs/` (pending cloud cleanup worker, `OrphanEnqueuer`), `internal/system/jobs/` (RBAC sync schedulers). A module only needs this folder when it owns long-running or scheduled background work. |

### Dependency rule (strict)

```
delivery, jobs → application → infra → domain
```

- `domain` MUST NOT import any other internal layer.
- `application` MUST NOT import `delivery` or `jobs`.
- `infra` MUST NOT import `application`, `delivery`, or `jobs`.
- `jobs` (when present) MAY import its own `application` and/or `domain` package, but MUST NOT import `delivery`. Treat `jobs` as a peer "entry point" to `delivery` — `delivery` is triggered by HTTP, `jobs` is triggered by a ticker/scheduler.
- Cross-domain dependencies (e.g. Auth needing RBAC) are injected as **interfaces** defined in the consuming domain and adapted in `internal/server/wire.go`.

---

## Constants Placement

- Domain-specific constants (TTLs, limits, status values unique to one domain) belong **inside** that domain package (e.g. `internal/auth/domain/token_ttl.go`, `internal/media/domain/bunny_status_codes.go`).
- **Cross-domain constants** belong in `internal/shared/constants/` — currently only 5 files:
  - `dbschema_name.go` — PostgreSQL table/relation names
  - `error_msg.go` — reusable error message strings + numeric caps
  - `media.go` — media upload limits, multipart constants
  - `permissions.go` — `AllPermissions` catalog
  - `register_http.go` — registration rate-limit HTTP header names
- Do NOT declare business constants inline inside service, infra, or delivery files.

---

## Error Handling

### Error definition

- Domain errors (not reused cross-domain) live in `internal/<domain>/domain/errors.go`.
- Shared/cross-domain sentinel errors live in `internal/shared/errors/`.
- Every user-facing error must have a numeric code in `internal/shared/errors/` (or `pkg/httperr`).

### Error propagation

- Application and infra layers return domain errors or wrap them.
- Delivery (handler) layer is the only place that maps errors to HTTP status codes and error codes.
- Never hardcode HTTP status or error codes in `application/` or `infra/`.

### GORM not-found mapping

Map `gorm.ErrRecordNotFound` to the shared `ErrNotFound` sentinel at the infra/repository boundary.

---

## API Patterns

- All JSON responses use the standard envelope via `internal/shared/response` — never raw `gin.H`.
- List endpoints use a `BaseFilter` struct that provides `page`, `per_page`, `sort_by`, `sort_order`, `search_by`, `search_data`.
- Fine-grained permission guards are applied **at the route level** using `middleware.RequirePermission`.

---

## Security Patterns

- JWT-based auth: `Authorization: Bearer <token>` header only (cookies are set for the browser but are not read by the middleware).
- Route security tiers:
  - **unauthenticated** — public endpoints (register, login, confirm, refresh, health)
  - **authenticated** — `AuthJWT` middleware validates the Bearer token
  - **permission-gated** — `RequirePermission` checks the JWT embedded permission set
  - **internal** — `RequireInternalAPIKey` for RBAC admin API
  - **system** — `RequireSystemAccessToken` for privileged operations

---

## Structured Logging (Uber Zap)

- **Bootstrap:** `main.go` calls `setting.Setup()` first, then `logger.InitFromSettings()`, then `defer logger.Sync()`. Only stdlib `log` is used before logger init.
- **Global logger:** `logger.InitFromSettings()` calls `zap.ReplaceGlobals`. Any package may use `zap.L()` or `zap.S()` after that.
- **Per-request fields:** `middleware.RequestLogger()` generates `X-Request-ID`, attaches it to `c.Request.Context()` via `logger.WithRequestID`. Handlers and services log with `logger.FromContext(ctx)` for automatic `request_id` inclusion.
- **Do not log HTTP bodies** (PII risk). Access log includes method, path, status, latency, bytes, IP, and `request_id`.
- **Config keys** (all optional): `LOG_LEVEL`, `LOG_FORMAT` (`json`/`console`), `LOG_FILE_PATH` (append-only NDJSON for Filebeat), `LOG_SERVICE_NAME`, `LOG_ENVIRONMENT`, `APP_VERSION`.

```go
// In a handler
logger.FromContext(c.Request.Context()).Info("created", zap.String("id", id))

// In a background job
zap.L().Info("tick", zap.String("job", "media-cleanup"))
```

---

## Data Access Patterns

- GORM as primary ORM. Raw SQL only when required (e.g. complex RBAC joins).
- Embedded SQL migrations via `golang-migrate`.
- PostgreSQL table names: defined once in `internal/shared/constants/dbschema_name.go`. All GORM `TableName()` methods and raw SQL reference these constants — no hardcoded string literals.

---

## Testing

Tests are **co-located** with their packages, not in a separate `tests/` directory.

| Convention | Rule |
|-----------|------|
| Package name | `<package>_test` (black-box) or `<package>` (white-box) |
| File naming | `<something>_test.go` adjacent to the code under test |
| Location | Same directory as the package being tested |

Current test files:

| File | Package |
|------|---------|
| `internal/media/application/batch_delete_test.go` | `application_test` |
| `internal/media/application/batch_upload_test.go` | `application_test` |
| `internal/media/infra/pipeline_test.go` | `infra_test` |
| `internal/media/infra/orphan_safety_test.go` | `infra_test` |
| `internal/media/infra/orphan_image_test.go` | `infra_test` |
| `internal/media/infra/bunny_webhook_test.go` | `infra_test` |
| `internal/media/delivery/server_owned_test.go` | `delivery_test` |
| `internal/shared/utils/webp_test.go` | `utils_test` |
| `internal/shared/logger/logger_test.go` | `logger_test` |

Run all tests:

```bash
go test ./...
```

---

## Linting

- `golangci-lint` is configured at repo root via `.golangci.yml`.
- **`revive file-length-limit`**: max 300 logical lines per `.go` file. Split oversized files by cohesive concern within the same package.
- **`funlen`**: max 30 lines / 25 statements per function. Extract unexported helpers when you touch a long function.
- When a function grows past `funlen`, prefer **unexported helpers in the same package** rather than moving logic to a different layer.

```bash
golangci-lint run
```

### Makefile compile targets (`Makefile`)

- **`make build`** — production compile with **`CGO_ENABLED=1`** (requires **`libvips-dev`**, **`libhdf5-dev`**, and **`pkg-config`** on Ubuntu when libvips pulls **matio**/HDF5; see CI workflow).
- **`make build-nocgo`** — pure Go compile when CGO/libvips are unavailable; at runtime, WebP encode may return **`9017`** (`ImageEncodeBusy`); other features are unaffected.

---

## Documentation Sync

When public JSON, DTOs, DB migrations, or persistence columns change for a documented feature, update every maintained doc referencing that feature. Minimum checklist:

1. `docs/modules/<domain>.md`
2. `docs/data-flow.md`
3. `docs/router.md`
4. `docs/modules.md`, `README.md`, `docs/architecture.md`
5. `docs/database.md` (if schema changed)
