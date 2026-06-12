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

Enforced by **`go-arch-lint`** (`make check-architecture`, config `.go-arch-lint.yml`) and **`depguard`** in `.golangci.yml`.

```
delivery, jobs → application → domain
infra          → domain (+ application only when infra implements an application-facing adapter — rare)
server/wire    → all layers (composition root)
```

- `domain` MUST NOT import `application`, `infra`, `delivery`, `jobs`, or framework/ORM packages listed in `.golangci.yml` (`gorm.io/gorm`, `encoding/json`, `database/sql/driver`, `gin`, `jwt`, etc.).
- `application` MUST NOT import `infra`, `delivery`, or `jobs`. It depends on **`domain` repository interfaces** and optional **`domain` ports** (e.g. `MediaGateway`, `SystemCrypto`).
- `infra` implements domain repositories and ports; MUST NOT import `application`, `delivery`, or `jobs`.
- `delivery` MUST NOT import `infra`. Handlers call `application` services and, when needed, **`domain` ports** injected alongside the service in `wire.go` (e.g. multipart parsing on `MediaGateway`).
- `jobs` (when present) MAY import its own `application` and/or `domain`, and MAY import `infra` for cloud deletes; MUST NOT import `delivery`.
- Cross-domain dependencies (e.g. Auth → RBAC permissions, Taxonomy → Media profile image) use **interfaces** on the consuming side, adapted in `internal/server/wire.go`.
- Do **not** add an `internal/*/entity` package or layer; persistence models (`userRow`, GORM tags, JSONB `Valuer`/`Scanner`) stay in **`infra`** only.

---

## Constants Placement

- Domain-specific constants (TTLs, limits, status values unique to one domain) belong **inside** that domain package (e.g. `internal/auth/domain/token_ttl.go`, `internal/media/domain/bunny_status_codes.go`).
- **Cross-domain constants** belong in `internal/shared/constants/`:
  - `dbschema_name.go` — PostgreSQL table/relation names
  - `error_msg.go` — reusable error message strings + numeric caps
  - `media.go` — media upload limits, multipart constants
  - `mq_topics.go` — LavinMQ topic exchange default + routing keys
  - `permissions.go` — `AllPermissions` catalog
  - `ratelimit.go` — shared rate-limit keys / caps
  - `register_http.go` — registration rate-limit HTTP header names
- Do NOT declare business constants inline inside service, infra, or delivery files.

---

## Error Handling

### Error definition

- Domain errors (not reused cross-domain) live in `internal/<domain>/domain/errors.go`.
- Shared/cross-domain sentinel errors live in `internal/shared/errors/`.
- Every user-facing error must have a numeric code in `internal/shared/errors/`. HTTP transport mapping uses `internal/shared/httperr` middleware and optional `HTTPError` values.

### Error propagation

- Application and infra layers return domain errors or wrap them.
- Delivery (handler) layer is the only place that maps errors to HTTP status codes and error codes.
- Never hardcode HTTP status or error codes in `application/` or `infra/`.

### GORM not-found mapping

Map `gorm.ErrRecordNotFound` to the shared `ErrNotFound` sentinel at the infra/repository boundary. Prefer `internal/shared/gormx.FirstWhere` for repeated `Where(...).First(...)` patterns.

### Domain ports (infra behind interfaces)

When `application` or `delivery` needs cloud SDKs, crypto, or multipart helpers that cannot live in `domain` entities:

| Port | Package | Infra adapter | Wired in |
|------|---------|---------------|----------|
| `MediaGateway` | `internal/media/domain/gateway.go` | `internal/media/infra/storage_gateway.go` | `server/wire.go` → `MediaService` + `delivery.Handler` |
| `SystemCrypto` | `internal/system/domain/ports.go` | `internal/system/infra/crypto_ports.go` | `server/wire.go` → `SystemService` |

Repository interfaces remain in `domain/repository.go`; ports cover **stateless infrastructure operations** that are not entity persistence.

### Auth persistence split (canonical)

| Concern | Layer | Files |
|---------|-------|-------|
| `User`, `RefreshTokenSessionMap`, session entry types | `domain/` | `user.go` — no GORM tags, no `encoding/json` for DB |
| GORM row + column tags | `infra/` | `user_model.go` (`userRow`), `toUserDomain` / `toUserRow` |
| JSONB `Value`/`Scan` for `refresh_token_session` | `infra/` | `gormjsonb.go` |
| Repositories | `infra/` | `user_repo.go`, `user_query.go` |

---

## API Patterns

- All JSON responses use the standard envelope via `internal/shared/response` — never raw `gin.H`.
- List endpoints use a `BaseFilter` struct that provides `page`, `per_page`, `sort_by`, `sort_order`, `search_by`, `search_data`.
- Fine-grained permission guards are applied **at the route level** using `middleware.RequirePermission`, typically through the shared route helper `internal/shared/utils.RoutePermission(...)` so delivery packages do not duplicate local permission-wrapper closures.

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

## LavinMQ / CloudAMQP (topic messaging)

Optional async messaging via [CloudAMQP LavinMQ](https://www.cloudamqp.com/docs/lavinmq-server.html) using the official Go client [`github.com/rabbitmq/amqp091-go`](https://www.cloudamqp.com/docs/go.html).

| Concern | Location |
|---------|----------|
| Config | `config/app*.yaml` → `lavinmq:` block; env `CLOUDAMQP_URL`, `LAVINMQ_EXCHANGE` (default `amq.topic`), `LAVINMQ_ENABLED` |
| Connection bootstrap | `internal/shared/mq/setup.go` — `SetupLavinMQ()` (non-fatal when disabled or URL empty, same as Redis warn-on-fail) |
| Publish | `mq.PublishTopic(ctx, routingKey, mq.Publishing{...})` |
| Subscribe | `mq.StartConsumers(ctx, mq.Subscription{RoutingKey, Handler, ...})` — one AMQP channel per consumer |
| Built-in consumers | `mq.StartDefaultConsumers` from `main.go` (health ping topic) |
| Topic routing keys | `internal/shared/constants/mq_topics.go` |

Domain modules that need async work should **publish/consume via `internal/shared/mq`** and add routing keys to `mq_topics.go` — do not open parallel AMQP clients in `infra/`.

```go
// Publish from application/infra when LavinMQ is enabled
_ = mq.PublishTopic(ctx, constants.TopicCoursePublished, mq.Publishing{
    Body:        payload,
    ContentType: "application/json",
})

// Register a consumer (e.g. from jobs/ or main via StartDefaultConsumers)
mq.StartConsumers(ctx, mq.Subscription{
    RoutingKey: constants.TopicMediaUploadCompleted,
    Handler:    myHandler,
})
```

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

### Audit timestamps (Unix seconds)

Since migration **`000011`**, `created_at` / `updated_at` / `deleted_at` are **`BIGINT`** Unix epoch **seconds** in Postgres and **`int64`** / **`*int64`** in Go domain and infra rows. JSON APIs expose them as **numbers**, not ISO/RFC3339 strings.

| Helper | Package | Use |
|--------|---------|-----|
| `NowUnix`, `PtrUnix`, `UnixOrZero` | `internal/shared/timex` | Current time and nullable epoch fields |
| `TouchCreatedUpdated`, `TouchUpdated` | `internal/shared/gormx` | Set audit fields on create/update in application/infra |
| `SoftDeleteWithAudit` | `internal/shared/gormx` | Set `deleted_at` + `updated_at` on soft delete |
| `ScopeActiveOnly` | `internal/shared/gormx` | `WHERE deleted_at IS NULL` for list/get |

Do not use GORM `autoCreateTime` / `autoUpdateTime` on audit columns — writes go through `timex` + `gormx` so DB and API stay aligned.

### CRUD APIs with soft delete

Convention for resources that support reversible deletion. Implemented on **taxonomy** (all five resources) and **auth users** (`/me` only). See **`docs/router.md`**, **`docs/database.md`**.

#### Semantics

| State | Rule |
|-------|------|
| Active row | `deleted_at IS NULL` |
| Soft-deleted row | `deleted_at > 0` (Unix epoch **seconds**) |
| Banned user (auth) | `banned_until IS NOT NULL AND banned_until > now()` — ban **expires at** that timestamp |
| Permanent disable (auth) | `is_disable = true` — separate from time-limited ban |

#### Routes

| Operation | Default | Variant |
|-----------|---------|---------|
| List / search | `GET /resources` — active only | `GET /resources/full` — includes soft-deleted |
| Get by id | `GET /resources/:id` — active only (404 if soft-deleted) | `GET /resources/:id/full` — when list-by-id is exposed |
| Delete | `DELETE /resources/:id` — soft delete | `DELETE /resources/:id/hard` — permanent GORM `Delete` |

Register **static** paths (`/full`, `/:id/hard`) **before** generic `/:id` routes in Gin.

#### Repository / infra

- List: `gormx.ScopeActiveOnly` unless `filter.IncludeDeleted` (taxonomy: `domain.TaxonomyFilter.IncludeDeleted`).
- GetByID (default): active-only — keeps PATCH/update safe.
- Soft delete: `gormx.SoftDeleteWithAudit` sets `deleted_at` and `updated_at`.
- Hard delete: GORM `db.Delete(&row{}, id)`; side effects (e.g. orphan image cleanup) **only** on hard delete.

#### Slug uniqueness (taxonomy)

Partial unique index `WHERE deleted_at IS NULL` (migration **`000012`**) so slugs can be re-created after soft delete.

#### Exceptions

| Area | Behavior |
|------|----------|
| **RBAC** | Junction row removes — hard delete only |
| **Auth `/me`** | No `GET /me/full`; login, refresh, `/me` reject deleted, disabled, and actively banned users |
| **Media** | Soft delete on `media_files`; route convention deferred |

#### Auth access guard

`AuthService.EnsureActiveUser` / `checkUserAccessible` in **`internal/auth/application/service_access.go`**. Wired into **`middleware.RequireActiveUser`** on every JWT-protected `/api/v1` route (after `AuthJWT`). Also used explicitly in login, refresh, and `/me` handlers.

---

## Testing

Tests are co-located with their packages.

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

## Linting and quality gate

Configuration: `.golangci.yml`, `.go-arch-lint.yml`, `Makefile`, `tools/layoutguard`.

| Check | Command | What it enforces |
|-------|---------|------------------|
| Format | `go fmt ./...` | Standard Go formatting |
| Static lint | `golangci-lint run` | `dupl` (per-package, threshold **60** — `.golangci.yml` `settings.dupl.threshold`), `depguard`, `revive`, `funlen`, `errcheck`, `staticcheck`, `govet`, `nolintlint` |
| Layer imports | `make check-architecture` | DDD boundaries (`go-arch-lint` v1.15+) |
| File placement | `make check-layout` | Go files only under allowed top-level dirs |
| Cross-package clones | `make check-dupl` | `dupl -t 60 internal` via Makefile `DUPL_THRESHOLD` (paths outside a single package) |
| Compile (no CGO) | `make build-nocgo` | CI-friendly build; WebP encode uses stub |
| Tests | `go test ./...` | All package tests |

**`revive` highlights:** `file-length-limit` max **600** lines (comments/blank lines skipped); `argument-limit` max **8** parameters — use small param structs when needed.

**`funlen`:** max **60** lines / **40** statements per function.

**`dupl`:** golangci only sees clones **within one package** (threshold **60**); cross-package similarity must pass `make check-dupl` (same threshold via `DUPL_THRESHOLD` in the root **`Makefile`**).

### Local gate (run before push)

```bash
cd be-mycourse
go fmt ./...
golangci-lint cache clean && golangci-lint run
make check-layout
make check-architecture
make check-dupl
make build-nocgo
go test ./...
```

CI (`.github/workflows/deploy-dev.yml` **`test`** job) runs the same layout/arch/dupl targets after `golangci-lint run`.

### Makefile compile targets

- **`make build`** — **`CGO_ENABLED=1`** (requires **libvips-dev**, **libhdf5-dev**, **pkg-config** on Ubuntu).
- **`make build-nocgo`** — pure Go; runtime WebP encode may return **`9017`** (`ImageEncodeBusy`).

---

## Documentation Sync

When **public API**, DTOs, DB migrations, or persistence columns change, update the docs that reference that feature.

Always regenerate ApiDog/Postman import after `docs/api_swagger.yaml` changes.

Minimum checklist when behavior or API **does** change:

1. `docs/modules/<domain>.md`
2. `docs/data-flow.md`
3. `docs/router.md`
4. `docs/modules.md`, `README.md`, `docs/architecture.md`
5. `docs/database.md` (if schema changed)
6. `ruby scripts/generate-apidog-postman.rb` (if request/response contracts changed)
