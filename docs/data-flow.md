# Data Flow Snapshot


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

## Primary Request Flow
1. HTTP request enters Gin router (`api/router.go`).
2. Global middleware executes (`httperr`, recovery, CORS, gzip, interceptors).
3. Group middleware executes (JWT/API key/system token/rate limit depending route group).
4. Handler validates input DTO and calls service.
5. Service executes business logic, persistence, and optional cache.
6. Unified response envelope is returned via `pkg/response`.

## End-to-End Flows (Current)

### Auth Register
- `POST /api/v1/auth/register` -> `api/v1/auth.go:register` -> `auth.Register` in **`services/auth/register_flow.go`** (pending create or unconfirmed resend); **`services/auth/register_email_send.go`** enforces lifetime cap + Brevo send + counter increment.
- Enforces **`users.registration_email_send_total`** lifetime cap (**15** successful sends while pending; next attempt **deletes** row -> **`410` / `4009`**) and Redis sliding window (**5 / 4h** per `user_id` -> **`429` / `4010`** + **`Retry-After`** + **`X-Mycourse-Register-Retry-After`**).
- Confirmation email via **`pkg/brevo`**. On success increments `registration_email_send_total` and keeps Redis window entry from the pre-send reservation (`services/cache/register_email_window.go`).
- **`GET /api/v1/auth/confirm`** -> `auth.ConfirmEmail` (**`services/auth/auth.go`** + **`services/auth/auth_confirm.go`** helpers): resets `registration_email_send_total`, deletes Redis window, clears login email cache.

### Auth Login
- `POST /api/v1/auth/login` -> `auth.Login` (`services/auth`).
- Reads cache (login invalid/email->user) then DB user lookup/password verify.
- Issues access/refresh tokens, persists refresh session JSONB (`users.refresh_token_session`), updates cache.

### Auth Refresh
- `POST /api/v1/auth/refresh` -> `auth.RefreshSession` (`services/auth`).
- Parses refresh token, validates session in DB JSONB, refreshes permission list, rotates token/session metadata.

### Me/Profile
- `GET /api/v1/me` (JWT) -> `auth.GetMe` (`services/auth`) -> **`entities.MeProfile`**; **`api/v1/me.go`** maps to **`dto.MeResponse`** via **`mapping.ToMeResponseFromProfile`**.
- Cache-first `me` fetch, fallback to DB + permission resolution, then cache write.

### Permission Check
- `middleware.RequirePermission` checks JWT embedded permission set.
- If absent, falls back to `rbac.UserHasAllPermissions` (`services/rbac`) -> DB permission resolution.

### System Sync
- `/api/system/*` routes (system token protected except login).
- Trigger immediate RBAC sync (`internal/rbacsync`) or start/stop in-memory periodic jobs (`internal/jobs/rbac`, `internal/jobs/media`, HTTP adapters in `internal/jobs/system`).

### Taxonomy CRUD
- `/api/v1/taxonomy/*` (JWT + permission protected) → taxonomy handlers → taxonomy services → `repository/taxonomy` (GORM repos).
- List endpoints apply shared filter parsing (`pkg/query/filter_parser.go`) with whitelist sorting/search.
- Mutations normalize taxonomy status via **`pkg/taxonomy.NormalizeTaxonomyStatus`** (`pkg/taxonomy/status.go`; `services/taxonomy` imports that package) before repository writes.
- Public responses are mapped via `pkg/logic/mapping` into DTO contracts (`CategoryResponse`, `CourseLevelResponse`, `TagResponse`).
- Persistence targets new taxonomy tables from migration `000002_taxonomy_domain.*`.

### Media Upload CRUD
- `/api/v1/media/files*` (JWT + permission protected) -> media handlers -> media services -> provider SDK/HTTP clients.
- **Single-file size cap:** `constants.MaxMediaUploadFileBytes` (**2 GiB**, defined in **`constants/error_msg.go`**). Oversize user-facing copy is **`constants.MsgFileTooLargeUpload`** (used for both JSON `message` via `errcode.DefaultMessage(FileTooLarge)` and `pkg/errors.ErrFileExceedsMaxUploadSize`). Handler rejects declared `multipart.FileHeader.Size` over cap (**413** + `errcode.FileTooLarge`); service uses `io.LimitReader(max+1)` before buffering so streams cannot exceed the cap silently. Missing `file` remains **400** + `BadRequest` (distinct error path).
- **Multipart parsing:** Gin engine `MaxMultipartMemory` (**64 MiB** in `api.InitRouter`) so large parts spill to temp disk during parse (see `docs/modules/media.md`).
- Service resolves kind/provider and metadata from server-owned inputs only:
  - provider comes from server config (`setting.MediaSetting.AppMediaProvider`) and is not accepted from client API params.
  - `kind`/`metadata` multipart text fields are ignored after backward-compat parse validation.
  - kind inference is server-side by MIME/extension; if inference is unknown and no configured provider exists, provider fallback is `Local`.
  - service delegates typed metadata inference to **`pkg/media.BuildTypedMetadata`** and merges only provider/server-derived values.
  - non-video file branch: B2 origin URL + Gcore CDN URL
  - video branch: Bunny Stream playback URL
  - local branch: reversible signed token URL (`/media/files/local/:token`)
- **Sub 11 upload pipeline (create/update):**
  - For **image files** (`pkg/media.IsImageMIMEOrExt`): acquire `utils.imageEncodeGate` slot, run `utils.EncodeWebP(payload)` (bimg/libvips, `CGO_ENABLED=1`), release slot. Payload, MIME, filename (`.webp`), and size updated before `uploadToProvider`. Encode failure → `ProviderError{Code: 9017}` → HTTP **503**.
  - For **non-image, non-video CREATE** files: first 16 bytes checked against extension + magic-byte denylist via `utils.IsExecutableUploadRejected`. Match → `ErrExecutableUploadRejected` → HTTP **400** + code **2004**.
- Persisted rows live in `media_files`: `000003_media_metadata`, `000004_media_orphan_safety` (`row_version`, `content_fingerprint`, `media_pending_cloud_cleanup`), **`000005_media_bunny_response_fields`** (`video_id`, `thumbnail_url`, `embeded_html`). Replace uploads may enqueue superseded cloud objects into `media_pending_cloud_cleanup`; `internal/jobs/media/media_pending_cleanup_scheduler.go` processes deletes asynchronously (`main.go` starts the job after `config.InitSystem()`).
- Media response is mapped through `pkg/logic/mapping` to `dto.UploadFileResponse` (public payload hides internal `provider`; **no `origin_url` in JSON** (Sub 12); Bunny parity top-level fields when populated — `docs/modules/media.md`).

### Media Video Status + Webhook
- `GET /api/v1/media/videos/:id/status` -> `api/v1/media/getVideoStatus` -> `services/media.GetVideoStatus` (returns **`entities.VideoProviderStatus`**) -> handler maps to **`dto.VideoStatusResponse`** -> Bunny `GET /library/{libraryID}/videos/{guid}`.
- Numeric Bunny status is normalized by **`pkg/media.BunnyStatusString(status)`** with `unknown` fallback for unsupported values.
- `POST /api/v1/webhook/bunny` is mounted outside auth/permission middleware; `api/v1/media/bunnyWebhook` verifies signature on raw bytes, then **`pkg/logic/mapping`** (`UnmarshalBunnyVideoWebhookRequestJSON`, `ValidateBunnyVideoWebhookRequest`) parses/validates the JSON before **`services/media.HandleBunnyVideoWebhook`**.
- Webhook applies metadata/duration sync when status matches finished (`constants.FinishedWebhookBunnyStatus`); **`ApplyBunnyDetailToMetadata`** refreshes **`video_id` / `thumbnail_url` / `embeded_html`** in JSON and ORM columns; idempotent when DB row missing.

### Orphan Image Cleanup Flow (Sub 07 + Sub 14 FK path)
- **URL-based (Sub 07):** `EnqueueOrphanImageCleanup` (**`internal/jobs/media`**) — used for legacy URL fields and JSONB URL harvesting.
  1. DB lookup via `repository/media.FileRepository.GetByURL` → uses stored provider/key.
  2. Fallback: **`pkg/media.ParseImageURLForOrphanCleanup`** parses URL by pattern (Bunny prefix or B2/CDN prefix from `MediaSetting`).
  3. Inserts `media_pending_cloud_cleanup` row for deferred worker deletion.
- **FK-based (Sub 14):** taxonomy categories and user avatars store **`image_file_id`** / **`avatar_file_id`** → `media_files.id`. On **replace** or **delete**, `EnqueueOrphanCleanupForMediaFileID` (**`internal/jobs/media`**) loads the row and enqueues the same pending cleanup pipeline (skips **Local** provider rows with no cloud object).
- Future JSONB domains: **`pkg/media.ScanJSONBForImageURLs(raw)`** collects URLs from nested JSONB payloads before cascade delete.

## Persistence Boundaries
- PostgreSQL via GORM and selected raw SQL (`services/rbac/rbac.go`).
- Redis optional cache (auth and me payload optimization).
- SQL migrations via embedded files and `golang-migrate`.

## Data-Risk Hotspots
- Permission staleness window when RBAC changes but client still uses old JWT.
- `role_permissions` full rebuild in sync can remove access immediately if constants are incomplete.
- Session JSONB map in `users` centralizes refresh sessions in a single row payload.
