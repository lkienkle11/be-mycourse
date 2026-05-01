# Router Snapshot


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Initialization
- Router is created in `api.InitRouter()`.
- `InitRouter` sets `router.MaxMultipartMemory = 64 << 20` (64 MiB) so large multipart bodies spill to disk during parse; per-file upload cap is still enforced in media handlers/services (`constants.MaxMediaUploadFileBytes` in **`constants/error_msg.go`**, 2 GiB).
- `main.go` runs `router.Run(":"+port)` after service/bootstrap initialization.

## Route Hierarchy
- Root middleware stack:
  - `httperr.Middleware()`
  - `httperr.Recovery()`
  - CORS
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
- `api/system/routes.go` -> system operations.
- `api/v1/routes.go` -> auth/me, internal RBAC, and mounts taxonomy + media route groups.
- `api/v1/taxonomy/routes.go` -> taxonomy CRUD endpoint registration (`levels`, `categories`, `tags`).
- `api/v1/media/routes.go` -> media upload endpoint registration (`/media/files` with GET/POST/PUT/DELETE/OPTIONS + local token decode + video status route) and webhook route mount used by no-filter lane. Response schema for list/get/create/update: **`dto.UploadFileResponse`** including optional Bunny fields **`video_id`**, **`thumbnail_url`**, **`embeded_html`** — `docs/modules/media.md`, `docs/api_swagger.yaml`.

## Media request contract notes
- `POST/PUT /api/v1/media/files` reads multipart text fields through helper binders, but `kind` and `metadata` are ignored by service business flow (server-owned policy).
- Effective kind/provider are resolved server-side (`ResolveMediaKindFromServer`, `ResolveUploadProvider`).

## Authorization Pattern
- Authentication middleware on group level.
- Fine-grained permission middleware (`RequirePermission`) is used at endpoint level, including taxonomy CRUD permissions.
