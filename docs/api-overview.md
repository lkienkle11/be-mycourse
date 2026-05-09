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

For **route-level detail** (handlers, contracts, shared packages): **[`docs/modules/taxonomy.md`](modules/taxonomy.md)** — categories, tags, course levels, **`pkg/taxonomy`** (`NormalizeTaxonomyStatus`); **[`docs/modules/media.md`](modules/media.md)** — files/videos, webhooks, **`pkg/media`** (resolver, metadata, multipart). **`docs/return_types.md`** and **`docs/api_swagger.yaml`** mirror JSON shapes where listed.

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
- `PATCH /me` — partial profile update; body supports **`avatar_file_id`** (UUID of an existing **`media_files`** row). Response uses nested **`avatar`** (`dto.MediaFilePublic`) instead of a raw URL string.
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
  - `OPTIONS /media/files/batch-delete`
  - `POST /media/files/batch-delete`
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
- Media `POST/PUT /api/v1/media/files` enforces: **at most 5** multipart parts per request (`files` / `file`); **per-part** max **2 GiB** (`constants.MaxMediaUploadFileBytes`); **aggregate** sum of all parts in one request **≤ 2 GiB** (`constants.MaxMediaMultipartTotalBytes`). Use errcodes **2003** (per part), **2005** (aggregate), **2006** (too many parts). `POST /api/v1/media/files/batch-delete` accepts up to **10** distinct `object_keys` (**2008** / **2009** when limits violated). Reverse proxies must allow **≥ 2G** request bodies on the API vhost (`docs/deploy.md`).
- Media metadata and kind are server-owned:
  - `kind` is inferred server-side from MIME/extension.
  - if kind cannot be inferred and no configured provider exists, provider fallback is `Local`.
  - response `metadata` uses typed contract `UploadFileMetadata` (not `any`) with zero-value defaults for unavailable fields.
  - Bunny videos: success `data` may include **`video_id`**, **`thumbnail_url`**, **`embeded_html`** on `UploadFileResponse` (`docs/modules/media.md`, `docs/return_types.md`, `docs/api_swagger.yaml`).
  - **No `origin_url`** on `UploadFileResponse` (Sub 12 — canonical B2/origin URL is server-only; not in public JSON — `docs/modules/media.md`).
- **Sub 11 upload policies:**
  - Image uploads are **converted to WebP** (bimg/libvips, `CGO_ENABLED=1`) before cloud upload. Concurrency gate: `constants.MaxConcurrentImageEncode` (4). Build requires `libvips-dev`. Encode failure → HTTP **503** + code **9017** (`ImageEncodeBusy`).
  - Non-image, non-video `POST /media/files` files whose extension or magic bytes match the executable/script denylist are rejected → HTTP **400** + code **2004** (`ExecutableUploadRejected`).

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
