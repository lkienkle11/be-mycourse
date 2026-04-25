# Data Flow Snapshot

## Primary Request Flow
1. HTTP request enters Gin router (`api/router.go`).
2. Global middleware executes (`httperr`, recovery, CORS, gzip, interceptors).
3. Group middleware executes (JWT/API key/system token/rate limit depending route group).
4. Handler validates input DTO and calls service.
5. Service executes business logic, persistence, and optional cache.
6. Unified response envelope is returned via `pkg/response`.

## End-to-End Flows (Current)

### Auth Register
- `POST /api/v1/auth/register` -> `api/v1/auth.go:register` -> `services.Register`.
- DB uniqueness check and user create in `models.DB`.
- Confirmation email side effect through `pkg/brevo`.

### Auth Login
- `POST /api/v1/auth/login` -> `services.Login`.
- Reads cache (login invalid/email->user) then DB user lookup/password verify.
- Issues access/refresh tokens, persists refresh session JSONB (`users.refresh_token_session`), updates cache.

### Auth Refresh
- `POST /api/v1/auth/refresh` -> `services.RefreshSession`.
- Parses refresh token, validates session in DB JSONB, refreshes permission list, rotates token/session metadata.

### Me/Profile
- `GET /api/v1/me` (JWT) -> `services.GetMe`.
- Cache-first `me` fetch, fallback to DB + permission resolution, then cache write.

### Permission Check
- `middleware.RequirePermission` checks JWT embedded permission set.
- If absent, falls back to `services.UserHasAllPermissions` -> DB permission resolution.

### System Sync
- `/api/system/*` routes (system token protected except login).
- Trigger immediate RBAC sync (`internal/rbacsync`) or start/stop in-memory periodic jobs (`internal/jobs`).

### Taxonomy CRUD
- `/api/v1/taxonomy/*` (JWT + permission protected) -> taxonomy handlers -> taxonomy services -> taxonomy repositories.
- List endpoints apply shared filter parsing (`pkg/query/filter_parser.go`) with whitelist sorting/search.
- Mutations normalize taxonomy status via `helper.NormalizeTaxonomyStatus` (`pkg/logic/helper/taxonomy_status.go`) before repository writes.
- Persistence targets new taxonomy tables from migration `000002_taxonomy_domain.*`.

### Media Upload CRUD
- `/api/v1/media/files*` (JWT + permission protected) -> media handlers -> media services -> provider SDK/HTTP clients.
- Service normalizes metadata and dispatches by provider:
  - non-video file branch: B2 origin URL + Gcore CDN URL
  - video branch: Bunny Stream playback URL
  - local branch: reversible signed token URL (`/media/files/local/:token`)
- Media descriptor is returned directly in response and is not persisted in local DB.

## Persistence Boundaries
- PostgreSQL via GORM and selected raw SQL (`services/rbac.go`).
- Redis optional cache (auth and me payload optimization).
- SQL migrations via embedded files and `golang-migrate`.

## Data-Risk Hotspots
- Permission staleness window when RBAC changes but client still uses old JWT.
- `role_permissions` full rebuild in sync can remove access immediately if constants are incomplete.
- Session JSONB map in `users` centralizes refresh sessions in a single row payload.
