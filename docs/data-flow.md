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
- Public responses are mapped via `pkg/logic/mapping` into DTO contracts (`CategoryResponse`, `CourseLevelResponse`, `TagResponse`).
- Persistence targets new taxonomy tables from migration `000002_taxonomy_domain.*`.

### Media Upload CRUD
- `/api/v1/media/files*` (JWT + permission protected) -> media handlers -> media services -> provider SDK/HTTP clients.
- **Single-file size cap:** `constants.MaxMediaUploadFileBytes` (**2 GiB**, defined in **`constants/error_msg.go`**). Oversize user-facing copy is **`constants.MsgFileTooLargeUpload`** (used for both JSON `message` via `errcode.DefaultMessage(FileTooLarge)` and `pkg/errors.ErrFileExceedsMaxUploadSize`). Handler rejects declared `multipart.FileHeader.Size` over cap (**413** + `errcode.FileTooLarge`); service uses `io.LimitReader(max+1)` before buffering so streams cannot exceed the cap silently. Missing `file` remains **400** + `BadRequest` (distinct error path).
- **Multipart parsing:** Gin engine `MaxMultipartMemory` (**64 MiB** in `api.InitRouter`) so large parts spill to temp disk during parse (see `docs/modules/media.md`).
- Service normalizes metadata and dispatches by provider:
  - provider comes from server config (`setting.MediaSetting.AppMediaProvider`) and is not accepted from client API params.
  - default provider resolution helper lives in `pkg/logic/helper/media_metadata.go` and generic conversion primitives are delegated to `pkg/logic/utils/parsing.go`.
  - service delegates metadata parsing/inference to helper layer; client metadata is optional and backend infers typed metadata from payload/provider outputs.
  - non-video file branch: B2 origin URL + Gcore CDN URL
  - video branch: Bunny Stream playback URL
  - local branch: reversible signed token URL (`/media/files/local/:token`)
- Persisted rows live in `media_files` (migration `000003_media_metadata`, extended by `000004_media_orphan_safety` with `row_version` + `content_fingerprint`). Replace uploads may enqueue superseded cloud objects into `media_pending_cloud_cleanup`; `internal/jobs/media_pending_cleanup_scheduler.go` processes deletes asynchronously (`main.go` starts the job after `config.InitSystem()`).
- Media response is mapped through `pkg/logic/mapping` to `dto.UploadFileResponse` (public payload hides internal provider field).

### Media Video Status + Webhook
- `GET /api/v1/media/videos/:id/status` -> `api/v1/media/getVideoStatus` -> `services/media.GetVideoStatus` -> Bunny `GET /library/{libraryID}/videos/{guid}`.
- Numeric Bunny status is normalized by `helper.BunnyVideoStatus.StatusString()` (`unknown` fallback for unsupported values).
- `POST /api/v1/webhook/bunny` is mounted outside auth/permission middleware and calls `services/media.HandleBunnyVideoWebhook`.
- Webhook applies metadata/duration sync when status matches finished (`constants.FinishedWebhookBunnyStatus`); idempotent when DB row missing.

### Orphan Image Cleanup Flow (Sub 07)
- Triggered when a business entity with an image URL field is deleted or has its URL replaced.
- `services/taxonomy.DeleteCategory` / `UpdateCategory` → `mediasvc.EnqueueOrphanImageCleanup(url)`.
- `EnqueueOrphanImageCleanup` (in `services/media/orphan_cleanup.go`):
  1. DB lookup via `repository/media.FileRepository.GetByURL` → uses stored provider/key.
  2. Fallback: `helper.ParseImageURLForOrphanCleanup` parses URL by pattern (Bunny prefix or B2/CDN prefix from `MediaSetting`).
  3. Inserts `media_pending_cloud_cleanup` row for deferred worker deletion.
- Future JSONB domains: `helper.ScanJSONBForImageURLs(raw)` collects URLs from nested JSONB payloads before cascade delete.

## Persistence Boundaries
- PostgreSQL via GORM and selected raw SQL (`services/rbac.go`).
- Redis optional cache (auth and me payload optimization).
- SQL migrations via embedded files and `golang-migrate`.

## Data-Risk Hotspots
- Permission staleness window when RBAC changes but client still uses old JWT.
- `role_permissions` full rebuild in sync can remove access immediately if constants are incomplete.
- Session JSONB map in `users` centralizes refresh sessions in a single row payload.
