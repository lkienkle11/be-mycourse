# Router Snapshot

## Initialization
- Router is created in `api.InitRouter()`.
- `main.go` runs `router.Run(":"+port)` after service/bootstrap initialization.

## Route Hierarchy
- Root middleware stack:
  - `httperr.Middleware()`
  - `httperr.Recovery()`
  - CORS
  - gzip

- Groups:
  - `/api/system`
    - `BeforeInterceptor`
    - `RateLimitSystemIP`
    - protected subgroup with `RequireSystemAccessToken`
  - `/api/v1`
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
- `api/v1/media/routes.go` -> media upload endpoint registration (`/media/files` with GET/POST/PUT/DELETE/OPTIONS and local token decode).

## Authorization Pattern
- Authentication middleware on group level.
- Fine-grained permission middleware (`RequirePermission`) is used at endpoint level, including taxonomy CRUD permissions.
