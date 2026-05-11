# Router Snapshot


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Initialization
- Router is created in `api.InitRouter()`.
- `InitRouter` sets `router.MaxMultipartMemory = constants.MediaMultipartParseMemoryBytes` (64 MiB) so large multipart bodies spill to disk during parse; per-part cap (`constants.MaxMediaUploadFileBytes`), per-request part count / aggregate cap (`MaxMediaFilesPerRequest`, `MaxMediaMultipartTotalBytes`), and streaming guards are enforced in media handlers/services (see **`docs/modules/media.md`**).
- `main.go` runs `router.Run(":"+port)` after service/bootstrap initialization.

## Route Hierarchy
- Root middleware stack:
  - `middleware.RequestLogger()` — structured access log + **`X-Request-ID`** propagation (see `docs/patterns.md`).
  - `httperr.Middleware()`
  - `httperr.Recovery()`
  - CORS (`api/router.go` — `ExposeHeaders` includes **`X-Token-Expired`**, **`Retry-After`**, **`X-Mycourse-Register-Retry-After`** so browser clients can read token-expiry and registration rate-limit hints cross-origin)
  - gzip

- Groups:
  - `/api/v1` (no-filter lane)
    - no `BeforeInterceptor`
    - mounted by `RegisterNoFilterRoutes` (currently webhook callbacks)
  - `/api/system`
    - `BeforeInterceptor`
    - `RateLimitSystemIP`
    - protected subgroup with `RequireSystemAccessToken`
  - `/api/v1` (standard lane)
    - `BeforeInterceptor`
    - authenticated subgroup: `RateLimitLocal` + `AuthJWT`
    - non-auth subgroup: `RateLimitLocal`
  - `/api/internal-v1`
    - `RateLimitLocal`
    - `BeforeInterceptor`
    - `RequireInternalAPIKey`

## Route Registration Units

For **per-domain contracts** (handlers, DTOs, provider policy), see **`docs/api-overview.md`** and the module deep-dives **`docs/modules/taxonomy.md`** / **`docs/modules/media.md`**.

- `api/system/routes.go` → system operations (`RegisterRoutes` receives the Gin group only; handlers use **`internal/appdb.Conn()`** for the shared GORM handle so **`api/`** does not import **`mycourse-io-be/models`** or **GORM** — see `docs/patterns.md` / depguard `restrict_api`).
- `api/v1/routes.go` -> **`GET`/`PATCH` `/me`**, me permissions, and mounts taxonomy + media route groups.
- `api/v1/taxonomy/routes.go` -> taxonomy CRUD endpoint registration (`levels`, `categories`, `tags`).
- `api/v1/media/routes.go` -> media upload endpoint registration (`/media/files` with GET/POST/PUT/DELETE/OPTIONS + local token decode + video status route) and webhook route mount used by no-filter lane. Response schema for list/get/create/update: **`dto.UploadFileResponse`** (no **`origin_url`** — Sub 12) including optional Bunny fields **`video_id`**, **`thumbnail_url`**, **`embeded_html`** — `docs/modules/media.md`, `docs/api_swagger.yaml`.

## Media request contract notes
- `POST/PUT /api/v1/media/files` reads multipart text fields through **`pkg/logic/mapping`** (`BindCreateFileMultipart` / `BindUpdateFileMultipart`), but `kind` and `metadata` are ignored by service business flow (server-owned policy).
- Effective kind/provider are resolved server-side (`ResolveMediaKindFromServer`, `ResolveUploadProvider`).

## Authorization Pattern
- Authentication middleware on group level.
- Fine-grained permission middleware (`RequirePermission`) is used at endpoint level, including taxonomy CRUD permissions.
