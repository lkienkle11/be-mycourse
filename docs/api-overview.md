# API Overview


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

## Base Route Groups
- `/api/system`: system authentication + RBAC sync/job controls.
- `/api/v1`: public and authenticated product APIs.
- `/api/internal-v1`: internal APIs protected by internal API key.

## Implemented Endpoint Inventory

### `/api/system`
- `POST /login`
- `POST /permission-sync-now`
- `POST /role-permission-sync-now`
- `POST /create-permission-sync-job`
- `POST /create-role-permission-sync-job`
- `POST /delete-permission-sync-job`
- `POST /delete-role-permission-sync-job`

### `/api/v1` (non-auth subgroup)
- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `GET /auth/confirm`
- `POST /auth/refresh`

### `/api/v1` (auth subgroup)
- `GET /me`
- `GET /me/permissions` (currently guarded with `RequirePermission(user:read)`).
- Taxonomy (admin CRUD):
  - `GET /taxonomy/levels`
  - `POST /taxonomy/levels`
  - `PATCH /taxonomy/levels/:id`
  - `DELETE /taxonomy/levels/:id`
  - `GET /taxonomy/categories`
  - `POST /taxonomy/categories`
  - `PATCH /taxonomy/categories/:id`
  - `DELETE /taxonomy/categories/:id`
  - `GET /taxonomy/tags`
  - `POST /taxonomy/tags`
  - `PATCH /taxonomy/tags/:id`
  - `DELETE /taxonomy/tags/:id`
- Media upload/files:
  - `OPTIONS /media/files`
  - `GET /media/files`
  - `POST /media/files`
  - `OPTIONS /media/files/:id`
  - `GET /media/files/:id`
  - `PUT /media/files/:id`
  - `DELETE /media/files/:id`
  - `OPTIONS /media/files/local/:token`
  - `GET /media/files/local/:token`
  - `GET /media/videos/:id/status`
  - Note: multipart text fields `kind` and `metadata` are accepted for backward-compat parsing only and ignored in create/update business flow (server-owned policy).
- Public webhook (registered before auth middleware):
  - `POST /webhook/bunny`

### `/api/internal-v1/rbac`
- Permissions CRUD: list/create/update/delete.
- Roles CRUD + set role permissions.
- User-role and user-direct-permission assignment/list/removal APIs.

## Upload / body limits (media)
- Media `POST/PUT /api/v1/media/files` enforces **2 GiB** max per `file` part (`constants.MaxMediaUploadFileBytes` in **`constants/error_msg.go`**); oversize JSON `message` / sentinel both use **`constants.MsgFileTooLargeUpload`**. HTTP **413** + app code **2003** when exceeded. Reverse proxies must allow **≥ 2G** request bodies on the API vhost (`docs/deploy.md`).
- Media metadata and kind are server-owned:
  - `kind` is inferred server-side from MIME/extension.
  - if kind cannot be inferred and no configured provider exists, provider fallback is `Local`.
  - response `metadata` uses typed contract `UploadFileMetadata` (not `any`) with zero-value defaults for unavailable fields.
  - Bunny videos: success `data` may include **`video_id`**, **`thumbnail_url`**, **`embeded_html`** on `UploadFileResponse` (`docs/modules/media.md`, `docs/return_types.md`, `docs/api_swagger.yaml`).

## Middleware/Auth Matrix
- Global: request/recovery/CORS/gzip.
- `/api/v1` auth subgroup: local rate limit + JWT auth.
- `/api/internal-v1`: local rate limit + internal API key middleware.
- `/api/system`: interceptor + system-IP rate limit + system access token (except `/login`).

## Gaps vs Planned E-learning Domains
- Taxonomy domain is now implemented in Phase 01.
- Media upload domain is now implemented in Phase sub 02 with unified file/video API branch.
- No course/lesson/enrollment/commerce CRUD endpoints implemented yet.
- Planned module docs (`docs/modules/*.md`) describe intended touchpoints in `api/v1`, `services`, `dto`, `models`, `migrations`.
