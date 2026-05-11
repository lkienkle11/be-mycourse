# MyCourse Backend

## Documentation Convention (Mandatory)

The `docs/` folder is the **primary and authoritative documentation source** for this project.

- **Before starting any task** (coding, planning, debugging, refactoring), read the relevant files in `docs/` first.
- If `docs/` already contains sufficient and up-to-date information → **reuse it directly** without re-running full discovery.
- If `docs/` is missing information or outdated → re-run discovery and **update `docs/` before proceeding**.
- Always sync `docs/` after completing any task that changes architecture, APIs, data flow, patterns, or reusable assets.
- `docs/reusable-assets.md` must be checked before proposing any new utility, type, or helper to avoid duplication.

| Doc | Contents |
|-----|----------|
| [`docs/architecture.md`](docs/architecture.md) | HTTP layers, directory map, `/api/v1` vs internal routes, GitNexus graph snapshot |
| [`docs/folder-structure.md`](docs/folder-structure.md) | Full directory tree with purpose of every folder and subfolder |
| [`docs/data-flow.md`](docs/data-flow.md) | How data moves through the system end-to-end |
| [`docs/api-overview.md`](docs/api-overview.md) | Full API surface: routes, handlers, request/response shapes |
| [`docs/logic-flow.md`](docs/logic-flow.md) | Control flow and execution paths for key operations |
| [`docs/router.md`](docs/router.md) | Routing structure and handler registration |
| [`docs/patterns.md`](docs/patterns.md) | Coding patterns and conventions (errors, constants, **`pkg/media`** / **`pkg/taxonomy`** / **`pkg/logic/utils`** layout, tests layout) |
| [`docs/dependencies.md`](docs/dependencies.md) | Key libraries, frameworks, and their relationships |
| [`docs/reusable-assets.md`](docs/reusable-assets.md) | All reusable utilities, types, DTOs, error codes, constants |
| [`docs/database.md`](docs/database.md) | Database schema, tables, migration history |
| [`docs/requirements.md`](docs/requirements.md) | Functional & non-functional requirements for all features |
| [`docs/sequence_diagrams.md`](docs/sequence_diagrams.md) | Mermaid sequence diagrams for every system flow |
| [`docs/return_types.md`](docs/return_types.md) | Go service return types and full JSON response shapes per API |
| [`docs/curl_api.md`](docs/curl_api.md) | Complete API reference with cURL examples and Postman scripts |
| [`docs/modules.md`](docs/modules.md) | Module responsibilities overview — implemented modules, planned modules, ownership boundaries, and testing layout |
| [`docs/modules/`](docs/modules/) | Per-domain module notes (auth, user, media, taxonomy, rbac, course, lesson, enrollment) |
| [`docs/modules/media.md`](docs/modules/media.md) | Media module deep-dive (`pkg/media`, webhooks, multipart limits) |
| [`docs/modules/taxonomy.md`](docs/modules/taxonomy.md) | Taxonomy module deep-dive (`pkg/taxonomy`, `services/taxonomy`) |

---

## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable **domain** types must be declared in **`pkg/entities`** (no `gorm` / `database/sql` imports there — same depguard rule as **`models/*.go`**).
- GORM/JSONB **column** types used on models: refresh-session JSONB **`Valuer`/`Scanner`** in **`pkg/gormjsonb/auth`**, carrier→domain map in **`pkg/logic/mapping/auth_refresh_session_mapping.go`** (**`ToRefreshTokenSessionEntity`**, Rule 14), GORM **`DeletedAt`** alias in **`models/deleted_at.go`** — not **`pkg/entities`**.
- Do not declare new reusable/domain types inline inside logic implementation files.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.
- **Postgres table names** for GORM / SQL live in **`constants/dbschema_name.go`**; **`dbschema/`** provides typed accessors (`RBAC`, `Media`, `Taxonomy`, `System`, `AppUser`) — see `IMPLEMENTATION_PLAN_EXECUTION.md` (Phase Sub 13).

Backend scaffold aligned to the monolith layout in `36.md` (inspired by `openedu-core`).

| Doc | Contents |
|-----|----------|
| [`docs/architecture.md`](docs/architecture.md) | HTTP layers, directory map, `/api/v1` vs internal routes, GitNexus graph snapshot |
| [`docs/patterns.md`](docs/patterns.md) | Coding patterns and conventions (errors, constants, **services file naming** vs `helper_` / `_helper`, **`pkg/media`** / **`pkg/taxonomy`** / utils, tests layout) |
| [`docs/deploy.md`](docs/deploy.md) | VPS + CI/CD runbook |
| [`docs/database.md`](docs/database.md) | Database schema, tables, migration history |
| [`docs/requirements.md`](docs/requirements.md) | Functional & non-functional requirements for all features |
| [`docs/sequence_diagrams.md`](docs/sequence_diagrams.md) | Mermaid sequence diagrams for every system flow |
| [`docs/return_types.md`](docs/return_types.md) | Go service return types and full JSON response shapes per API |
| [`docs/curl_api.md`](docs/curl_api.md) | Complete API reference with cURL examples and Postman scripts |
| [`docs/modules.md`](docs/modules.md) | Module responsibilities overview — implemented modules, planned modules, ownership boundaries |
| [`docs/modules/`](docs/modules/) | Per-domain notes (auth, user, media, taxonomy, rbac, course, lesson, enrollment) |
| [`docs/modules/media.md`](docs/modules/media.md) | Media: B2+Gcore+Bunny, `media_files`, **`video_id` / `thumbnail_url` / `embeded_html`**, webhook + status, policy in `media_resolver.go`; **per-part and aggregate 2 GiB**, **≤5 parts/request**, batch delete **≤10 keys**, multipart + proxy (`docs/deploy.md`) |
| [`docs/modules/taxonomy.md`](docs/modules/taxonomy.md) | Taxonomy: categories / tags / course levels; shared **`pkg/taxonomy`** (`NormalizeTaxonomyStatus`) + `services/taxonomy` |
| `pkg/entities/` | Shared **domain** structs (no `gorm` / `database/sql` / **no funcs**); depguard **`restrict_models_pkg_entity_schema_only`** — see **`docs/patterns.md`** |
| `pkg/gormjsonb/auth/` | JSONB **`Valuer`/`Scanner`** for **`users.refresh_token_session`** (**`RefreshTokenSessionMap`** defined type) — see **`docs/patterns.md`** |
| `pkg/logic/mapping/auth_refresh_session_mapping.go` | **`ToRefreshTokenSessionEntity`** — JSONB carrier → **`entities.RefreshTokenSessionMap`** (Rule 14) — see **`docs/patterns.md`** |
| `models/deleted_at.go` | GORM **`DeletedAt`** type alias (**`gorm.DeletedAt`**) — see **`docs/patterns.md`** |
| [`tests/`](tests/) | **All test code** (unit/module-level/integration) — place test packages, fixtures, and shared harnesses here (see **Testing** below). |

## Quick Start

1. Ensure Redis is running.
2. Copy `.env.example` to `.env`, set `STAGE` if needed, and fill the keys used in `config/app.yaml` or `config/app-<STAGE>.yaml`:
   - `supabase.url` placeholders → `SUPABASE_URL`
   - `SUPABASE_SERVICE_ROLE_KEY`
   - `SUPABASE_DB_URL` (pooler or direct)
   - `APP_BASE_URL` — public base URL of this server, used in outgoing emails (no trailing slash), e.g. `https://api.mycourse.io`
   - `CORS_ALLOWED_ORIGINS` — comma-separated list of allowed frontend origins, e.g. `http://localhost:3000,https://mycourse.io`
   - `CLI_REGISTER_NEW_SYSTEM_USER` — optional; when `true`/`1`/`yes`/`y`/`on`, after DB connect the binary runs the privileged-user registration CLI and exits (see `docs/architecture.md`).
   - **Logging (Uber Zap):** optional env keys `LOG_LEVEL`, `LOG_FORMAT` (`json` \| `console`), `LOG_FILE_PATH` (append-only NDJSON file + `zapcore.NewTee` for ELK/Filebeat), `LOG_SERVICE_NAME`, `LOG_ENVIRONMENT`, `APP_VERSION`, `LOG_REDIRECT_STDLOG` — wired via `config/app*.yaml` → `setting.LogSetting` → `pkg/logger` (see `docs/patterns.md`).
3. Run:

```bash
go mod tidy
go run .
```

### RBAC sync

- `go run ./cmd/syncpermissions` upserts `permissions.permission_name` by `permission_id` from `constants.AllPermissions` (extra DB rows are not deleted).
- `go run ./cmd/syncrolepermissions` deletes all `role_permissions` rows and repopulates them from `constants.RolePermissions` (roles resolved by name; `permission_id` is taken from tags as-is).
- **`POST /api/system/login`** then **`POST /api/system/*-sync-now`** / **`create-*-sync-job`** (12h in-memory tickers) — see `docs/architecture.md` and `api/system/routes.go`.

4. Verify:

```bash
curl http://localhost:8080/api/v1/health
```

## Testing

- **Module tests** (integration flows, black-box tests against `mycourse-io-be`, shared fixtures, or any test code you want outside production packages): add packages under **`tests/`** at the repository root (alongside `api/`, `services/`, …). `go test ./...` from the repo root includes those packages once they contain `*_test.go` files.
- **All test code** (including unit, integration, black-box, fixtures, and shared harnesses) **MUST** be placed under repository root **`tests/`**.
- On-disk pointer: [`tests/README.md`](tests/README.md).
- Canonical convention text: [`docs/patterns.md`](docs/patterns.md) (mirrored in [`docs/patterns.md`](docs/patterns.md)) and [`docs/requirements.md`](docs/requirements.md) (NFR on test layout).

### Local verification (fmt / vet / test / lint / build)

```bash
gofmt -w .
go vet ./...
go test ./...
golangci-lint run
make build-nocgo
```

`golangci-lint` enables **revive** `file-length-limit` (**300** logical lines per file) and **funlen** (**30** lines / **25** statements per function). Oversized **files** are split by cohesive concern in the same package (see **`docs/patterns.md`**). Some legacy functions still exceed **funlen**; extract small unexported helpers when you touch them, or treat cleanup as a follow-up pass.

Use `make build` when `CGO_ENABLED=1` and `libvips-dev` are available (see Makefile).

### CI deploy (`master`)

Pushing to **`master`** runs `.github/workflows/deploy-dev.yml`: build **`mycourse-io-be-dev`** in Actions, on the server **copy the current dev binary to `bin/mycourse-io-be-dev.prev`** (only place that can run **before** `rsync` overwrites the file), **`rsync`** the new binary **to the same path** `bin/mycourse-io-be-dev`, then run **`scripts/pm2-reload-with-binary-rollback.sh`**. That script **alone** snapshots **`ecosystem.config.cjs` → `ecosystem.config.cjs.prev`**, pulls **only** that file from **`origin/master`**, **`pm2 reload`**, waits for **`GET /api/v1/health`** (and PM2 **`max_restarts` exhaustion** via `pm2 jlist`), then **full `git pull`** on success; on failure it restores **binary + ecosystem** from the two **`.prev`** files. No separate “`.new`” staging file. Secrets: `SSH_PRIVATE_KEY`, `SSH_HOST`, `SSH_USER`, **`DEPLOY_PATH_DEV`**. Runbook: [`docs/deploy.md` — Appendix C](docs/deploy.md#appendix-c--cicd-with-github-actions).

---

## CORS

CORS is configured via the `CORS_ALLOWED_ORIGINS` environment variable — a **comma-separated list** of allowed origins.

```env
# .env (local / dev)
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173,http://localhost:5174

# .env.staging / .env.prod
CORS_ALLOWED_ORIGINS=https://mycourse.io,https://www.mycourse.io
```

The value is read by `pkg/setting` at startup, split on `,`, trimmed, and passed to `github.com/gin-contrib/cors`.
If the variable is empty or unset, it falls back to `http://localhost:3000`.

Allowed methods: `GET POST PUT PATCH DELETE OPTIONS`  
Allowed headers: `Origin Content-Type Authorization X-API-Key X-Refresh-Token X-Session-Id`  
Exposed headers: `X-Token-Expired`  
Credentials: enabled (`AllowCredentials: true`)

| Custom header | Direction | Purpose |
|---|---|---|
| `Authorization` | request | Bearer access token for all protected endpoints |
| `X-Refresh-Token` | request | Refresh JWT sent to `POST /api/v1/auth/refresh` |
| `X-Session-Id` | request | Session ID sent to `POST /api/v1/auth/refresh` |
| `X-Token-Expired` | response | `"true"` when a 401 is caused by an **expired** access JWT (`jwt.ErrTokenExpired`). Not set for `missing bearer token` — see `docs/modules/auth.md` and `middleware/auth_jwt.go`. |

---

## API Response Format

All responses are JSON objects. There are **two envelope shapes** depending on the endpoint.

### Standard response (all endpoints except `/health`)

```json
{
  "code":    0,
  "message": "ok",
  "data":    <value>
}
```

| Field     | Type                      | Description                                                           |
|-----------|---------------------------|-----------------------------------------------------------------------|
| `code`    | `number`                  | Custom app code. `0` = success. Non-zero = error (see error codes).   |
| `message` | `string`                  | Human-readable status or error message.                               |
| `data`    | `null \| string \| number \| boolean \| array \| object \| PaginatedData` | Response payload. `null` on errors or operations with no return value. |

### Health response (`GET /api/v1/health`)

```json
{
  "code":    0,
  "message": "ok",
  "status":  "ok"
}
```

| Field     | Type     | Description              |
|-----------|----------|--------------------------|
| `code`    | `number` | Always `0` when healthy. |
| `message` | `string` | Always `"ok"`.           |
| `status`  | `string` | Always `"ok"`.           |

---

### Paginated data (`data` as `PaginatedData`)

When an endpoint returns a paginated list, `data` is a **PaginatedData** object:

```json
{
  "code":    0,
  "message": "ok",
  "data": {
    "result": [ ... ],
    "page_info": {
      "page":        1,
      "per_page":    20,
      "total_pages": 5,
      "total_items": 98
    }
  }
}
```

| Field                    | Type                     | Description                             |
|--------------------------|--------------------------|-----------------------------------------|
| `data.result`            | `null \| string \| number \| boolean \| array \| object` | The actual page of records. |
| `data.page_info.page`        | `number` | Current page number (1-based).          |
| `data.page_info.per_page`    | `number` | Number of items per page.               |
| `data.page_info.total_pages` | `number` | Total number of pages.                  |
| `data.page_info.total_items` | `number` | Total number of items across all pages. |

---

### Error response

Error responses use the same standard envelope with a non-zero `code` and `data: null`:

```json
{
  "code":    4002,
  "message": "Invalid email or password",
  "data":    null
}
```

For validation errors, `data` may contain field-level details:

```json
{
  "code":    2001,
  "message": "Validation failed",
  "data": {
    "details": [
      { "field": "email", "message": "email is required" }
    ]
  }
}
```

---

### Application error codes (`code`)

| Code  | Constant             | Meaning                                        |
|-------|----------------------|------------------------------------------------|
| `0`   | `Success`            | Operation completed successfully               |
| `1001`| `InvalidJSON`        | Request body is not valid JSON                 |
| `2001`| `ValidationFailed`   | Request validation failed                      |
| `2002`| `ValidationField`    | Per-field validation error (used in `details`) |
| `3001`| `BadRequest`         | Bad request                                    |
| `3002`| `Unauthorized`       | Unauthorized                                   |
| `3003`| `Forbidden`          | Forbidden                                      |
| `3004`| `NotFound`           | Resource not found                             |
| `3005`| `Conflict`           | Conflict (e.g. duplicate resource)             |
| `3006`| `TooManyRequests`    | Rate limit exceeded                            |
| `4001`| `EmailAlreadyExists` | Email address is already registered            |
| `4002`| `InvalidCredentials` | Invalid email or password                      |
| `4003`| `WeakPassword`       | Password does not meet strength requirements   |
| `4004`| `EmailNotConfirmed`  | Email address has not been confirmed yet       |
| `4005`| `UserDisabled`       | Account has been disabled                      |
| `4006`| `InvalidConfirmToken`| Invalid or expired confirmation token          |
| `4007`| `InvalidSession`     | Session string unknown, missing, or UUID mismatch |
| `4008`| `RefreshTokenExpired`| Session has expired — re-login required        |
| `4009`| `RegistrationAbandoned` | Pending registration removed — confirmation email lifetime limit |
| `4010`| `RegistrationEmailRateLimited` | Too many confirmation emails in the sliding window |
| `4011`| `ConfirmationEmailSendFailed` | Confirmation email could not be sent (upstream) |
| `9001`| `InternalError`      | Internal server error                          |
| `9998`| `Panic`              | Unhandled panic (internal server error)        |
| `9999`| `Unknown`            | Unknown error                                  |

---

### Go helper — `pkg/response`

Use the helpers in `pkg/response` to write responses from any handler or middleware. Never write raw `gin.H` envelopes in handlers.

```go
// Success responses
response.Health(c)                              // GET /health only
response.OK(c, "ok", data)                     // HTTP 200
response.Created(c, "created", data)           // HTTP 201
response.OKPaginated(c, "ok", rows, pageInfo)  // HTTP 200 + pagination

// Error responses (without aborting middleware chain)
response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "bad input", nil)

// Error responses (abort middleware chain — use in middleware)
response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "not authenticated", nil)
```

#### Optional headers and cookies

Every helper accepts an optional `response.Options` as its **last** argument.
Use it to attach extra response headers or set additional cookies alongside the JSON body.
Both fields are plain `map[string]string`.

```go
response.OK(c, "ok", data, response.Options{
    Headers: map[string]string{
        "X-Request-ID": requestID,
        "X-Trace-ID":   traceID,
    },
    Cookies: map[string]string{
        "session_hint": "abc123",
    },
})
```

**Cookie defaults** when set via `Options.Cookies`:

| Attribute  | Default value |
|------------|---------------|
| `Path`     | `/`           |
| `MaxAge`   | `0` (session) |
| `Domain`   | _(empty)_     |
| `Secure`   | `false`       |
| `HttpOnly` | `true`        |

> For cookies that need full control (custom `MaxAge`, `Domain`, `SameSite`, `Secure` tied to run-mode, etc.) call `c.SetCookie(...)` or `c.SetSameSite(...)` directly **before** or **after** the response helper — it is always safe to mix both approaches in the same handler (see `setAuthCookies` in `api/v1/auth.go` for an example).

Omitting `Options` entirely produces identical behaviour to the previous API.

---

## GET List API — Query Filter Params

Every GET list endpoint uses a DTO that **embeds `dto.BaseFilter`**. This guarantees
a consistent set of pagination, sorting, and search query parameters across all list APIs.

### Standard query parameters

| Param         | Type     | Default | Description                                                                 |
|---------------|----------|---------|-----------------------------------------------------------------------------|
| `page`        | `number` | `1`     | Page number (1-based). Values ≤ 0 are treated as `1`.                       |
| `per_page`    | `number` | `20`    | Items per page. Values ≤ 0 default to `20`; values > `100` are capped at `100`. |
| `sort_by`     | `string` | —       | Field name to sort by (e.g. `created_at`, `name`). Handler must whitelist allowed columns. |
| `sort_order`  | `string` | `asc`   | Sort direction. Accepted: `asc`, `desc`. Any other value defaults to `asc`. |
| `search_by`   | `string` | —       | Field name to search in. Handler decides which columns are searchable.       |
| `search_data` | `string` | —       | Search term. Applied only when `search_by` is also provided.                |

All params are **optional**. A request with none of them returns page 1 with 20 items in ascending order.

### Example request

```
GET /api/v1/users?page=2&per_page=10&sort_by=created_at&sort_order=desc&search_by=name&search_data=john
```

### Defining a custom filter DTO

Every filter struct **must embed `dto.BaseFilter`**. Add domain-specific fields after the embed:

```go
type UserFilter struct {
    dto.BaseFilter               // required — provides the 6 standard fields
    Role   string `form:"role"`
    Status string `form:"status"`
}
```

### Binding in a handler

```go
func listUsers(c *gin.Context) {
    var q dto.UserFilter
    if err := c.ShouldBindQuery(&q); err != nil {
        response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
        return
    }

    rows, total, err := services.ListUsers(services.UserListParams{
        Offset:     q.GetOffset(),      // (page-1) * per_page
        Limit:      q.GetPerPage(),     // per_page with defaults/cap applied
        SortBy:     q.SortBy,          // whitelist before passing to DB
        SortOrder:  q.GetSortOrder(),  // "asc" | "desc"
        SearchBy:   q.SearchBy,
        SearchData: q.SearchData,
        Role:       q.Role,
    })
    if err != nil { ... }

    response.OKPaginated(c, "ok", rows, response.PageInfo{
        Page:       q.GetPage(),
        PerPage:    q.GetPerPage(),
        TotalPages: (total + q.GetPerPage() - 1) / q.GetPerPage(),
        TotalItems: total,
    })
}
```

### BaseFilter helper methods

| Method           | Returns  | Description                                          |
|------------------|----------|------------------------------------------------------|
| `GetPage()`      | `int`    | Page number, defaulting to `1`.                      |
| `GetPerPage()`   | `int`    | Page size, defaulting to `20`, capped at `100`.      |
| `GetOffset()`    | `int`    | SQL `OFFSET` = `(GetPage()-1) * GetPerPage()`.       |
| `GetSortOrder()` | `string` | `"asc"` or `"desc"`, never an invalid value.         |
| `HasSearch()`    | `bool`   | `true` when both `search_by` and `search_data` are set. |
| `HasSort()`      | `bool`   | `true` when `sort_by` is set.                        |

---

## Project Structure

- `main.go`: startup flow (settings, db, cache, migrate, bootstrap, queue, router).
- `api/`: router, route groups (`public`, `api/v1`, `api/internal-v1`), API config.
- `middleware/`: request interceptor for auth/permission/tenant hooks.
- `services/`, `services/cache/`, `services/media/`, `dto/`: business layer; `services/cache` holds Redis helpers (e.g. auth `/me` + login invalid cache), and `services/media` orchestrates the unified upload flow for non-video files and videos while shared **media** helpers live in **`pkg/media`** (same package as provider clients), taxonomy status normalization in **`pkg/taxonomy`**, and generic cross-feature primitives in **`pkg/logic/utils`**. Typed media metadata is inferred in backend (image/video/document), provider is selected from server config (`setting.MediaSetting.AppMediaProvider`), and public media response is mapped via `pkg/logic/mapping` to `dto.UploadFileResponse` (provider is not exposed; **no `origin_url` in JSON** — Sub 12; Bunny parity fields **`video_id`**, **`thumbnail_url`**, **`embeded_html`** when populated — see `docs/modules/media.md`). `dto.BaseFilter` is the mandatory base for all GET list query-param DTOs.
- `models/`, `migrations/`: persistence layer (GORM models + SQL migrations).
- `pkg/cache_clients/`: Redis client bootstrap (used for auth profile + login negative cache — see `docs/modules/auth.md`).
- `pkg/entities/file.go`: shared media `File` entity descriptor used by media service responses, including base `FileMetadata` plus typed `ImageMetadata`, `VideoMetadata`, and `DocumentMetadata`.
- `queues/`: async layer placeholder (RabbitMQ intentionally excluded).
- `pkg/response`: **unified API response envelope** — use this in all handlers.
- `pkg/errcode`: numeric application error codes and their default messages.
- `constants/`: RBAC IDs, roles, domain enums; **`constants/error_msg.go`** holds reusable **error message / sentinel strings** (and small related numeric caps). If the same text is used in **`pkg/errcode/messages.go`** and in a sentinel (`errors.New`), define it **once** in `error_msg.go` and reference it from `messages.go` (see `MsgFileTooLargeUpload` + `FileTooLarge`). Numeric JSON `code` values stay in `pkg/errcode/codes.go`.
- `pkg/httperr`: Gin middleware for centralised error handling (panic recovery, validation).
- `pkg/setting`: YAML config with per-stage files and `.env` map substitution.
- `pkg/envbool`: parsing common “truthy” environment variable values (`true`, `1`, `yes`, …) for feature flags.
- `config/`: `app.yaml` + `app-<STAGE>.yaml` and env examples.
- `tests/`: **all test code** (unit/module-level/integration) and harnesses (not application runtime code).
- `tracing/`, `runtime/`: observability and runtime placeholders.
