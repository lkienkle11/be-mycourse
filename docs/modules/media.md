# Media Module

## Overview

The media module (`internal/media/`) provides a unified API surface for file and video uploads with cloud storage providers. It handles:

- **Non-video files** — upload to Cloudflare R2; public URL via R2 custom domain
- **Videos** — upload and stream via BunnyCDN Stream
- **Local** — reversible signed URL token (dev/fallback)

Provider selection is **server-side only** (`setting.MediaSetting.AppMediaProvider`) — never accepted from client request params.

---

## Directory Layout

```
internal/media/
├── domain/
│   ├── media.go                   # File entity, upload types, OpenedUploadPart
│   ├── gateway.go                 # MediaGateway port (cloud upload, metadata, multipart, webhooks)
│   ├── repository.go              # FileRepository + PendingCleanupRepository interfaces
│   ├── errors.go                  # ProviderError only (sentinels live in shared/errors)
│   ├── bunny_status_codes.go      # Bunny numeric video status constants
│   ├── bunny_webhook.go           # Bunny webhook payload types
│   └── meta_keys.go               # JSON metadata key constants (video_id, thumbnail_url, etc.)
├── application/
│   ├── service.go                 # MediaService — CreateFiles, UpdateFileBundle, delete/list
│   └── service_upload_helpers.go  # Upload pipeline (prepareNormalizedUploadPart), batch helpers
├── infra/
│   ├── storage_gateway.go         # StorageGateway — implements domain.MediaGateway
│   ├── repos.go                   # GormFileRepository + GormPendingCleanupRepository
│   ├── provider_runtime.go        # Cloud bootstrap, CloudClients, RequireInitialized
│   ├── provider_r2.go             # R2 upload/delete/public URL
│   ├── provider_bunny.go          # Bunny Stream upload/get/status + webhook signature
│   ├── provider_local.go          # Local signed-URL encode/decode (no hardcoded secret fallback)
│   ├── media_storage_mime.go      # Server-side MIME canonicalization for R2 Content-Type
│   ├── media_classification.go    # Kind/provider/MIME rules, profile image acceptance, list filters
│   ├── media_entity_metadata.go   # Typed metadata, upload entity build, object-key resolution
│   ├── multipart.go               # Multipart collect/validate/open/close
│   ├── stored_object_delete.go    # Provider-routed cloud delete
│   └── media_replace_policy.go    # Superseded-object cleanup policy
├── delivery/
│   ├── handler.go                 # HTTP handlers (injected MediaGateway for multipart/metadata)
│   ├── webhook_handler.go         # Bunny webhook (signature via MediaGateway)
│   ├── routes.go                  # RegisterRoutes + RegisterWebhookRoutes
│   ├── dto.go                     # FileFilterRequest (embeds utils.BaseFilter), response DTOs
│   ├── mapping.go                 # Domain → DTO mapping, multipart bind helpers
│   └── server_owned_test.go       # delivery_test
└── jobs/
    ├── enqueue.go                 # OrphanEnqueuer (FK-based cleanup queue)
    ├── cleanup_job.go             # Scheduler, batch worker, metrics, constants
    └── duration_backfill_scheduler.go
```

### Layering

- **`application.MediaService`** and **`delivery.Handler`** must not import `internal/media/infra`.
- All R2/Bunny/multipart/webhook operations go through **`domain.MediaGateway`**, implemented by **`infra.StorageGateway`**, constructed in **`internal/server/wire.go`** and passed to both service and handler.
- WebP encoding remains in **`internal/shared/utils/`** (CGO / stub), invoked from application upload paths.
- HTTP upload flows use **`CreateFiles`** and **`UpdateFileBundle`** only; legacy single-file service methods and unused multipart DTO structs were removed.
- List pagination reuses **`utils.BaseFilter`** (`GetPage` / `GetPerPage`); batch-delete key validation uses **`utils.ValidateUniqueTrimmedStrings`**.
- Active-row lookups use **`gormx.FirstWhere`**; list queries use **`gormx.ScopeActiveOnly`**.
- Provider/kind constants come from **`internal/shared/constants/media.go`** only (no duplicate exports in `domain/media.go`).
- Bunny **Storage** SDK wiring was removed; only R2 + Bunny **Stream** + Local are initialized at runtime.
- Local signed URLs require **`LOCAL_FILE_URL_SECRET`** (or `setting.MediaSetting.LocalFileURLSecret`); there is no hardcoded secret fallback. Missing secret returns **`ErrDependencyNotConfigured`** on upload and local decode paths — never an empty URL.

---

## API Surface

All routes below are under `/api/v1`. Authenticated routes require `Authorization: Bearer <token>` and the indicated permission.

| Method | Path | Auth | Permission | Purpose |
|--------|------|------|-----------|---------|
| OPTIONS | `/media/files` | JWT | — | CORS preflight |
| GET | `/media/files` | JWT | `media_file:read` | List files (paginated) |
| GET | `/media/files/cleanup-metrics` | JWT | `media_file:read` | Deferred cleanup counters |
| POST | `/media/files` | JWT | `media_file:create` | Upload 1–5 file parts |
| OPTIONS | `/media/files/batch-delete` | JWT | — | CORS preflight |
| POST | `/media/files/batch-delete` | JWT | `media_file:delete` | Delete up to 10 files by object key |
| OPTIONS | `/media/files/:id` | JWT | — | CORS preflight |
| GET | `/media/files/:id` | JWT | `media_file:read` | Get file detail |
| PUT | `/media/files/:id` | JWT | `media_file:update` | Bundle update (1–5 parts) |
| DELETE | `/media/files/:id` | JWT | `media_file:delete` | Delete single file |
| OPTIONS | `/media/files/local/:token` | JWT | — | CORS preflight |
| GET | `/media/files/local/:token` | JWT | `media_file:read` | Decode local signed URL token |
| GET | `/media/videos/:id/status` | JWT | `media_file:read` | Bunny video processing status |
| POST | `/webhook/bunny` | **None** | — | Bunny Stream callback (no-filter lane) |

### List files (`GET /media/files`)

Query parameters (all optional unless noted):

| Param | Values | Default | Notes |
|-------|--------|---------|-------|
| `page` | int ≥ 1 | `1` | Page number |
| `per_page` | int 1–100 | `20` | Page size |
| `provider` | `S3`, `GCS`, `B2`, `R2`, `Bunny`, `Local` | — | Storage provider filter |
| `kind` | `FILE`, `VIDEO` | — | Overridden when `category` is set |
| `search` | string | — | Filename search (`ILIKE %term%`), not ID/object key search |
| `category` | `image`, `document`, `video` | — | Tab filter: forces `kind` and applies MIME/extension rules (`IsImageMIMEOrExt` for images; document extensions for documents) |
| `sort_by` | `created_at`, `updated_at`, `filename`, `size_bytes` | `created_at` | Whitelisted column only |
| `sort_order` | `asc`, `desc` | `desc` | Sort direction |

**Access filter (server-side, not a query param):** authenticated caller's `user_id` scopes the result set to **own files + public files** only. Rows with **`user_id` NULL** (legacy, pre-backfill) are **excluded** from list until owner backfill completes. Private files owned by other users are excluded.

**List fail-closed:** `MediaService.ListFiles` returns **`ErrMediaAccessDenied`** when `ViewerUserID` is empty — access scoping is enforced in the service layer, not only in the HTTP handler.

`DELETE /media/files/:id` uses **`:id` = `object_key`**, not the row UUID.
Provider routing for single delete is resolved from the persisted `media_files` row first:

1. Use stored `provider` (fallback to stored `kind` -> default provider when provider is empty).
2. Use stored `bunny_video_id` when present.
3. Only when no active row exists for `object_key`, fallback to legacy metadata-based inference (`video_guid` / `bunny_video_id`).

---

## Permissions

| Permission ID | Permission Name |
|--------------|----------------|
| P26 | `media_file:read` |
| P27 | `media_file:create` |
| P28 | `media_file:update` |
| P29 | `media_file:delete` |

**Default role grants** (`internal/system/application/roles_permission.go`; sync via `go run ./cmd/syncrolepermissions`):

| Role | Media permissions |
|------|-------------------|
| **sysadmin** / **admin** | P26–P29 |
| **instructor** | P26–P29 |
| **learner** | P26–P29 (added in code — no migration; operator syncs via system route) |

Route-level JWT permission is still required. **Row-level access** (owner + visibility) is enforced in the media service on list/get/update/delete — see **Ownership and visibility** below.

---

## Ownership and visibility

Migration **`000030_media_owner_visibility`** adds:

| Column | Values | Default |
|--------|--------|---------|
| `user_id` | `users.id` of uploader | set on every new upload from JWT `user_id` |
| `visibility` | `private`, `public` | **`private`** |

**List (`GET /media/files`):** returns rows where **`user_id` = caller** or **`visibility` = `public`**. Legacy rows with **`user_id` NULL** are **not listed** (transition until owner backfill + `NOT NULL` constraint). Other users' **private** files never appear in the media picker popup.

**Get / update / delete:** caller must be the owner when `user_id` is set and the row is **private**. **Public** rows are readable by anyone with P26; mutate (update/delete) still requires owner match when `user_id` is set. Rows with **`user_id` NULL** (legacy): **read** may still succeed when the caller knows the `object_key` (e.g. existing course references); **update/delete are denied** until owner is backfilled.

**Get (`GET /media/files/:id`):** requires an **active** `media_files` row for the `object_key`. There is **no URL synthesis fallback** when the row is missing or soft-deleted — the handler returns **not found** so row-level access cannot be bypassed via storage/CDN keys alone.

**Upload (`POST /media/files`):** optional multipart field `visibility` (`private` \| `public`); omitted → **`private`**. Server sets `user_id` from JWT — never from client body.

**R2 object key (new uploads):** `{user_code}/{8digits}-{sanitized-filename}` where `user_code` comes from JWT `ctx_user_code`. Bunny Stream and Local providers keep existing key rules; **`user_id`** / **`visibility`** are still persisted on the row.

**Out of scope (later):** repathing legacy flat R2 keys, **backfilling `user_id` on old rows**, and migrating `user_id` to **`NOT NULL`** after backfill completes.

---

## Upload Pipeline

### Single file upload (`POST /media/files`)

1. Bind multipart form fields (1–5 `files` parts; legacy `file` field accepted for single part).
2. Validate per-part size ≤ `MaxMediaUploadFileBytes` (2 GiB).
3. Validate aggregate size ≤ `MaxMediaMultipartTotalBytes` (2 GiB).
4. Validate part count ≤ `MaxMediaFilesPerRequest` (5).
5. For each part concurrently (up to `MaxConcurrentMediaUploadWorkers = 5`):
   a. **Trusted MIME routing** — `MIMEForUploadRouting` (`detectTrustedMIME` via `github.com/gabriel-vasile/mimetype` on bytes). Multipart `Content-Type` from the client is never used. When detection is empty or blocked, routing falls back to filename extension only (`ResolveMediaKindFromServer` / `IsImageMIMEOrExt`).
   b. **Kind and provider resolve** — `ResolveMediaKindFromServer` + `ResolveUploadProvider` use the routing MIME (content-aware when detection succeeds).
   c. **Executable check** (non-image, non-video only): reject if extension or magic bytes match denylist → HTTP 400 / code 2004.
   d. **WebP encoding** (images only): `IsImageMIMEOrExt` uses the same routing MIME; acquire encode gate, run `EncodeWebP` via bimg/libvips (CGO), update payload/filename/MIME/size.
   e. **Storage MIME canonicalization** — `CanonicalStorageMIME` + `applyStorageMIMEPolicy` for final DB `mime_type` and R2 `PutObject.ContentType` (see **R2 Object Content-Type** below).
   f. **Provider upload**: R2 (non-video) or Bunny Stream (video).
   g. **Metadata inference**: typed `ImageMetadata`, `VideoMetadata`, or `DocumentMetadata` inferred server-side from final MIME/extension.
6. Persist to `media_files` (DB create).
7. Return array of `UploadFileResponse` (one entry per part).

### Bundle update (`PUT /media/files/:id`)

- First part updates the row at `:id`.
- Additional parts create new rows.
- Response is an array `[updated, ...created]`.
- Supports `reuse_media_id`, `expected_row_version`, `skip_upload_if_unchanged` for optimistic concurrency.
- Conflict → HTTP 409.

### Batch delete (`POST /media/files/batch-delete`)

- JSON body: `{ "object_keys": ["key1", "key2"] }`.
- Max 10 distinct keys; duplicates rejected; missing key → validation error (all-or-nothing).

---

## WebP Image Encoding (Sub 11)

All image uploads are **synchronously converted to WebP** before upload to the storage provider.

- Implementation: `bimg`/libvips (CGO). Requires `CGO_ENABLED=1` and **`libvips-dev`**, **`libhdf5-dev`**, **`pkg-config`** at build time on stacks where libvips links **matio** (Ubuntu **.pc** resolution).
- Concurrency: bounded by a semaphore in `internal/shared/utils/` (`AcquireEncodeGate` / `ReleaseEncodeGate`); cap = `MaxConcurrentImageEncode` (4).
- `CGO_ENABLED=0` builds: stub returns `ErrImageEncodeBusy` → HTTP 503 / code 9017.
- After encoding: `payload`, `filename` (`.webp`), `mime` (`image/webp`), and `sizeBytes` are updated before provider upload.

---

## R2 Object Content-Type

Non-video uploads stored on Cloudflare R2 must carry the correct S3 object `ContentType` so browsers and CDNs serve files with the right MIME (e.g. `image/webp` for encoded images).

**Server-side MIME (single source for routing + storage):** multipart `Content-Type` from the client is never trusted. `internal/media/infra/media_storage_mime.go` exposes three layers used by `prepareNormalizedUploadPart`:

1. **`MIMEForUploadRouting(payload, filename)`** — runs first. Uses `detectTrustedMIME` (`github.com/gabriel-vasile/mimetype` on bytes) to drive **kind**, **provider route**, **WebP encode decision**, and executable checks. Returns detected MIME when safe; otherwise `""` so `ResolveMediaKindFromServer` / `IsImageMIMEOrExt` fall back to filename extension only.
2. **`detectTrustedMIME(payload, filename)`** — magic-number detector (direct dependency in `go.mod`). Classifies OOXML, archives, video, images, PDF, plain text.
3. **`applyStorageMIMEPolicy(detected, filename, kind)`** — final storage policy after routing/encode:
   - **WebP (images):** after server encode, require WebP magic bytes (`RIFF` + `WEBP`) → `image/webp`; otherwise `application/octet-stream`.
   - **Blocked active types** (`text/html`, `application/javascript`, `image/svg+xml`, …) → `application/octet-stream`.
   - **OOXML mismatch:** `application/zip` with `.docx/.xlsx/.pptx` extension → `application/octet-stream`.
   - **Video (`kind=VIDEO`):** require detected `video/*`; otherwise `application/octet-stream`.
   - **Extension/content mismatch** (e.g. HTML bytes with `.pdf` extension) → `application/octet-stream`.
   - **Unknown / empty detection** → `application/octet-stream`.

`CanonicalStorageMIME` runs **after** kind is known and WebP encode completes; it orchestrates `detectTrustedMIME` + `applyStorageMIMEPolicy` for DB `mime_type` and R2 metadata.

- **Create and update** (`POST /media/files`, `PUT /media/files/:id`): canonical MIME is stored in metadata under `domain.MediaMetaKeyMimeType` (`mime_type`).
- **`UploadR2`** (`internal/media/infra/provider_r2.go`): sets `s3.PutObjectInput.ContentType` from canonical metadata only (blocked-MIME filter); defaults to `application/octet-stream` when metadata is absent — no extra `mime.TypeByExtension` fallback layer.
- Applies to all R2 providers (`R2`, legacy `B2`, `S3`, `GCS`) routed through `UploadToProvider` → `UploadR2`.
- DB `media_files.mime_type` and R2 object `ContentType` stay aligned for new uploads and replacements.

---

## Executable Denylist (Sub 11)

Non-image, non-video file uploads are checked against an extension + magic-byte denylist:

- **Extensions:** `.exe .msi .dmg .app .deb .rpm .sh .bash .zsh .fish .bat .cmd .com .ps1 .vbs .jse .scr .pif .jar .war .ear .dll .so .dylib .elf`
- **Magic bytes:** PE/MZ, ELF, Mach-O variants, shebang (`#!`)
- Rejected files → HTTP 400 / code 2004

---

## Bunny Integration

### Video upload

After the Bunny PUT request succeeds, `GetBunnyVideoByID` is called.
`ApplyBunnyDetailToMetadata` merges the Bunny detail payload into the
in-memory `raw` metadata map and that map is then **serialised into the
`media_files.metadata_json` JSONB column** via
`serializeMergedMetadataJSON` (see `internal/media/infra/media_upload_entity.go`).
On every read, `rowToFile` re-derives the typed `Metadata` sub-object from
that same JSONB blob, so the API response (`metadata`) is always consistent
regardless of whether the row was just written or loaded from the database.
The raw blob itself stays server-side — only the typed view is exposed to
clients.
The flat scalar columns are also populated for fast lookups:

| Source key in `metadata_json` | Flat column          | Notes                                  |
|--------------------------------|----------------------|----------------------------------------|
| `video_guid` / `bunny_video_id` | `bunny_video_id`     | Bunny GUID                             |
| `bunny_library_id`             | `bunny_library_id`   | Bunny library ID                       |
| `video_id`                     | `video_id`           | Numeric ID when available, else GUID   |
| `thumbnail_url`                | `thumbnail_url`      | See "Thumbnail URL" below              |
| `embeded_html`                 | `embeded_html`       | iframe embed snippet                   |
| `direct_play_url`              | `direct_play_url`    | Direct play page on player.mediadelivery.net |
| `hls_playlist_url`             | `hls_playlist_url`   | CDN HLS master playlist                |
| `preview_animation_url`        | `preview_animation_url` | CDN animated preview WebP           |
| `length`                       | `duration`           | Seconds (int)                          |
| `video_provider`               | `video_provider`     | `bunny_stream`                         |
| `r2_bucket_name`               | `r2_bucket_name`     | R2 bucket name                         |

All other Bunny fields are persisted **only** inside `metadata_json` so the
schema does not need to grow per-field columns:

- **Video telemetry:** `width`, `height`, `length`, `framerate`, `rotation`,
  `bitrate`, `video_codec`, `output_codecs`, `audio_codec`
- **Encoding state:** `encode_progress`, `available_resolutions`,
  `storage_size`, `has_mp4_fallback`, `has_original`,
  `has_high_quality_preview`, `jit_encoding_enabled`
- **Thumbnail bits:** `thumbnail_filename`, `thumbnail_blurhash`,
  `thumbnail_count`
- **Descriptive:** `title`, `description`, `date_uploaded`, `views`,
  `is_public`, `category`, `collection_id`, `original_hash`

Because Bunny may not have finished transcoding when the upload returns,
`width`/`height`/`length` may be `0` immediately after upload. The
**Bunny webhook** repopulates them when transcoding completes.

### Bunny response field mapping

Bunny Stream's actual JSON response was verified from production logs.
The following field names are the canonical ones (verified May 2026):

| Our struct field          | Bunny JSON key          | Notes                                          |
|---------------------------|-------------------------|------------------------------------------------|
| `GUID`                    | `guid`                  |                                                |
| `VideoLibraryID`          | `videoLibraryId`        |                                                |
| `BunnyNumericID`          | `id`                    | Numeric ID (not always returned)               |
| `Title`                   | `title`                 |                                                |
| `Length`                  | `length`                | Seconds (float64)                              |
| `Framerate`               | `framerate`             |                                                |
| `Width` / `Height`        | `width` / `height`      |                                                |
| `OutputCodecs`            | `outputCodecs`          | e.g. `"x264"` — used as `video_codec`          |
| `AvailableResolutions`    | `availableResolutions`  | e.g. `"360p,480p,720p,1080p"`                  |
| `ThumbnailFileName`       | `thumbnailFileName`     | e.g. `"thumbnail.jpg"` — used to build URL     |
| `ThumbnailBlurhash`       | `thumbnailBlurhash`     |                                                |
| `EncodeProgress`          | `encodeProgress`        |                                                |
| `StorageSize`             | `storageSize`           | Bytes (int64)                                  |
| `HasMP4Fallback`          | `hasMP4Fallback`        |                                                |
| `HasOriginal`             | `hasOriginal`           |                                                |
| `HasHighQualityPreview`   | `hasHighQualityPreview` |                                                |
| `Bitrate` (legacy)        | `bitrate`               | Bunny rarely returns this — kept for forward   |
|                           |                         | compatibility, never overwrites valid data     |
| `VideoCodec` (legacy)     | `videoCodec`            | Same as above                                  |
| `AudioCodec` (legacy)     | `audioCodec`            | Same as above                                  |
| `ThumbnailURL` (legacy)   | `thumbnailUrl`          | Same as above                                  |

### Thumbnail URL resolution

Bunny does NOT return a full thumbnail URL. It returns only
`thumbnailFileName` (e.g. `"thumbnail.jpg"`). The full URL is built from a
configurable CDN pull-zone hostname:

```
https://{MEDIA_BUNNY_STREAM_CDN_HOSTNAME}/{videoGuid}/{thumbnailFileName}
```

`MEDIA_BUNNY_STREAM_CDN_HOSTNAME` (e.g. `vz-abcdef12-3456.b-cdn.net`)
maps to `setting.MediaSetting.BunnyStreamCDNHostname`. When the hostname
is empty the `thumbnail_url` column simply stays blank — the embed iframe
URL (`embeded_html`) still works because it is built from
`BunnyStreamBaseURL` + `BunnyStreamLibraryID` + `guid`.

The resolution priority inside `EnrichBunnyVideoDetail` is:

1. `thumbnailUrl` (if Bunny ever populates it directly)
2. `defaultThumbnailUrl` (legacy alias)
3. `{cdn_hostname}/{guid}/{thumbnailFileName}` (the canonical case today; filename
   defaults to `thumbnail.jpg` when Bunny omits `thumbnailFileName`)

> **Regression fix (Jun 2026).** Upload metadata parsing used `fmt.Sprintf("%v", nil)`
> for optional Bunny keys, which persisted the literal string `"<nil>"` into
> `thumbnail_url`. Parsing now uses `utils.StringFromRaw` plus
> `SanitizeMetadataURL`, and webhook refresh overwrites columns via
> `ApplyBunnyStreamFileColumns`.

### Bunny webhook (`POST /webhook/bunny`)

Mounted on the **no-filter lane** (no JWT, no rate limit).

1. Validate Bunny signature v1 on raw request body.
   - Headers required: `X-BunnyStream-Signature-Version: v1`, `X-BunnyStream-Signature-Algorithm: hmac-sha256`, `X-BunnyStream-Signature: <hex>`
   - Signature: `hex(HMAC-SHA256(rawBody, signingSecret))`
   - Signing secret: `setting.MediaSetting.BunnyStreamReadOnlyAPIKey` (fallback: `BunnyStreamAPIKey`)
2. Parse and validate JSON payload (`VideoLibraryId`, `VideoGuid`, `Status` 0..10).
3. Handle by status:
   - `Finished (3)` and `Resolution finished (4)`: load row, fetch Bunny detail via
     `GetBunnyVideoByID`, **merge metadata additively** (never overwrite valid
     prior data with zero values), call `ApplyBunnyStreamFileColumns` to persist
     delivery URLs (`thumbnail_url`, `direct_play_url`, `hls_playlist_url`,
     `preview_animation_url`, `embeded_html`), update DB. Column values are only
     overwritten when the resolved URL is non-empty (legacy `"<nil>"` placeholders
     are sanitised on read and replaced on the next successful webhook).
   - `Failed (5)` and `PresignedUploadFailed (8)`: mark row `FAILED` (idempotent).
   - Other statuses: accepted and intentionally ignored (idempotent callbacks).

> **Persistence note (regression fix, May 2026).** The repository method
> `UpsertByObjectKey` used to call `Assign(struct) + FirstOrCreate(struct)`,
> which made GORM compile the UPDATE from the struct and silently skip every
> zero-value field. That meant the Bunny webhook could correctly compute
> `row.Duration = 190` but the SQL written to PostgreSQL still kept
> `duration = 0`. The repository now performs an **explicit
> `Updates(map[string]any{...})`** with every editable column listed by
> name — including `duration`, `metadata_json`, `status`, `thumbnail_url`,
> and `row_version + 1` — so an idempotent webhook callback always rewrites
> the full row. The same fix protects the create path: when no row exists
> the repository falls back to `db.Create(row)`. The column map is
> exercised by `TestBuildUpsertUpdateColumns_persistsWebhookFields` and
> `TestBuildUpsertUpdateColumns_zeroValuesStillPresent` in
> `internal/media/infra/upsert_columns_test.go`.

---

## Response Contract (`UploadFileResponse`)

Public fields returned in API responses:

| Field | Notes |
|-------|-------|
| `url` | Public distribution URL |
| `object_key` | Storage object key (`{user_code}/…` for new R2 uploads) |
| `user_id` | Uploader UUID (omitted when empty) |
| `display_name` | Uploader display name from `users.display_name` (omitted when empty; populated on list/get via JOIN; on upload responses enriched from JWT for the current uploader). **Email is not exposed** on media list/get responses. |
| `visibility` | `private` or `public` |
| `metadata` | **Typed** sub-object (`UploadFileMetadata`). Normalised shape for image / video / document. See "Typed `metadata` shape" below. **This is the only metadata sub-object exposed to clients.** |
| `bunny_video_id` | Bunny video GUID |
| `bunny_library_id` | Bunny library ID |
| `video_id` | Numeric Bunny ID or GUID |
| `thumbnail_url` | Built from `BunnyStreamCDNHostname` + `guid` + `thumbnailFileName` (defaults to `thumbnail.jpg` when Bunny omits the filename); falls back to `thumbnailUrl` / `defaultThumbnailUrl` if Bunny ever populates them |
| `embeded_html` | Escaped `<iframe ...>` embed HTML |
| `direct_play_url` | Bunny direct play page: `https://player.mediadelivery.net/play/{libraryId}/{guid}` |
| `hls_playlist_url` | HLS master playlist on CDN: `https://{cdn}/{guid}/playlist.m3u8` |
| `preview_animation_url` | Animated preview WebP on CDN: `https://{cdn}/{guid}/preview.webp` |
| `duration` | Video duration in seconds (persisted on `media_files.duration`; sourced from Bunny webhook / backfill, not resolved at course read time) |

**Video duration backfill:** When Bunny encoding finishes but `duration` was still `0`, `applyBunnyWebhookFinishedStatus` persists Bunny `length` into `media_files.duration`. A **delayed** one-shot job (`StartVideoDurationBackfillJob`, ~2 min after process start) backfills stale rows without blocking HTTP startup. `ListFiles` / `GetFile` may enqueue the same sync **asynchronously** (deduped by `bunny_video_id`); responses are never blocked on Bunny. Course outline reads only the DB row.

| `row_version` | Optimistic concurrency version (Sub 06) |
| `content_fingerprint` | SHA-256 hex of file bytes (Sub 06) |

**`origin_url` and the full raw metadata blob are NOT in the public API response.**
The DB column `metadata_json` is populated with rich provider data (Bunny
telemetry, thumbnail blurhash, encode progress, available resolutions, …)
for server-side use only — it backs the typed `metadata` field above and is
queryable via SQL for analytics and orphan cleanup, but the untyped map is
never serialised onto the response.

### Typed `metadata` shape

The typed sub-object is derived from the stored `metadata_json` blob on
every read (see `rowToFile` in `internal/media/infra/repos.go`). Both
write paths (CREATE / UPDATE) and read paths (LIST / GET) reuse the same
builder (`BuildTypedMetadata`) so the shape is always consistent. The
write path also mirrors these exact typed keys back into `metadata_json`
via `ApplyTypedMetadataToRaw`; for videos this means `duration_seconds`
is persisted in JSONB, not only returned in memory.

| JSON key            | Type     | Populated for |
|---------------------|----------|---------------|
| `size_bytes`        | int64    | All files     |
| `width_bytes`       | int      | Image / video |
| `height_bytes`      | int      | Image / video |
| `mime_type`         | string   | All files     |
| `extension`         | string   | All files     |
| `duration_seconds`  | float64  | Video         |
| `bitrate`           | int      | Video         |
| `fps`               | float64  | Video         |
| `video_codec`       | string   | Video         |
| `audio_codec`       | string   | Video         |
| `has_audio`         | bool     | Video         |
| `is_hdr`            | bool     | Video         |
| `page_count`        | int      | Document      |
| `has_password`      | bool     | Document      |
| `archive_entries`   | int      | Archive       |

### Server-side `metadata_json` (storage only)

Even though the API only returns the typed `metadata` field above, the
`media_files.metadata_json` JSONB column persists the full provider blob so
operations like analytics, debugging, and re-derivation of typed fields
remain possible. The column exists from `000003_media_metadata`; migration
`000008_media_metadata_json_storage` reasserts the JSONB default for existing
environments, backfills the typed metadata keys (`duration_seconds`,
`width_bytes`, `height_bytes`, `fps`, ...), and adds the GIN index
`idx_media_files_metadata_json_gin` for server-side JSONB queries. For Bunny
videos this includes:

```jsonc
{
  "video_guid": "7312c208-054f-413f-82b2-3666a271f4f4",
  "bunny_video_id": "7312c208-054f-413f-82b2-3666a271f4f4",
  "bunny_library_id": "650694",
  "video_provider": "bunny_stream",
  "video_id": "7312c208-054f-413f-82b2-3666a271f4f4",
  "thumbnail_url": "https://vz-xxxxxxxx-xxxx.b-cdn.net/.../thumbnail.jpg",
  "direct_play_url": "https://player.mediadelivery.net/play/650694/7312c208-054f-413f-82b2-3666a271f4f4",
  "hls_playlist_url": "https://vz-xxxxxxxx-xxxx.b-cdn.net/7312c208-054f-413f-82b2-3666a271f4f4/playlist.m3u8",
  "preview_animation_url": "https://vz-xxxxxxxx-xxxx.b-cdn.net/7312c208-054f-413f-82b2-3666a271f4f4/preview.webp",
  "embeded_html": "<iframe ...></iframe>",
  // Provider-native Bunny keys
  "length": 190,
  "width": 1920,
  "height": 1080,
  "framerate": 23.976,
  "video_codec": "x264",
  // Typed UploadFileMetadata keys persisted by ApplyTypedMetadataToRaw
  "duration_seconds": 190,
  "width_bytes": 1920,
  "height_bytes": 1080,
  "fps": 23.976,
  "size_bytes": 586177908,
  "mime_type": "video/mp4",
  "extension": ".mp4",
  // Bunny-only enrichment kept for server-side use
  "output_codecs": "x264",
  "available_resolutions": "360p,480p,720p,240p,1080p",
  "encode_progress": 100,
  "storage_size": 586177908,
  "has_mp4_fallback": true,
  "has_original": true,
  "has_high_quality_preview": true,
  "thumbnail_filename": "thumbnail.jpg",
  "thumbnail_blurhash": "WA8gHS%1IpxYt6of0gRkxZR+WCf6WBNHWCofoeayt7xaoJjsWVfk",
  "thumbnail_count": 95,
  "title": "Avicii - The Nights - YouTube.mp4",
  "date_uploaded": "2026-05-12T08:54:03.606",
  "original_hash": "4132A4196E...8848D"
}
```

---

## Orphan Cleanup (Sub 06 + Sub 14)

When a file is **replaced** or a parent row is **deleted**, the superseded cloud object is queued for deferred deletion:

- **FK-based (Sub 14):** taxonomy `course_topics.image_file_id`, `course_outcomes.image_file_id`, or `users.avatar_file_id` replacement → `EnqueueOrphanCleanupForFileID` (loads row by **`media_files.id`**, then enqueues provider/object key)
- **Object-key lookup:** `EnqueueOrphanCleanupByObjectKey` — accepts a stored **`object_key`** only (no raw URL parsing). Returns `false` when no active row matches.

The background cleanup worker in `jobs/cleanup_job.go` processes `media_pending_cloud_cleanup` rows asynchronously. Local provider rows are skipped.

**Retry semantics:** On transient cloud-delete failure, `MarkFailed` with a `nextRunAt` backoff keeps the row `PENDING`, increments `attempt_count`, and reschedules. After `MediaCleanupMaxAttempts`, the row is marked `FAILED` permanently and **`attempt_count` is incremented** on that final transition so DB metrics match the worker's attempt accounting.

---

## Upload Size and Transport

| Limit | Value | Code |
|-------|-------|------|
| Per-part cap | 2 GiB (`MaxMediaUploadFileBytes`) | 2003 FileTooLarge |
| Aggregate per-request | 2 GiB (`MaxMediaMultipartTotalBytes`) | 2005 MediaMultipartTotalTooLarge |
| Parts per request | 1–5 (`MaxMediaFilesPerRequest`) | 2006 MediaTooManyFilesInRequest |
| Batch delete | max 10 keys | 3001 BadRequest |

Gin `MaxMultipartMemory` = 64 MiB (set in `internal/server/router.go`). Large parts spill to temp disk during parse. **Reverse proxies must allow ≥ 2 GiB request bodies** — see `docs/deploy.md`.

---

## Provider Errors

| Code | Name | HTTP | Condition |
|------|------|------|-----------|
| 9019 | `R2BucketNotConfigured` | 500 | R2 bucket not configured |
| 9011 | `BunnyStreamNotConfigured` | 500 | Bunny Stream not configured |
| 9012 | `BunnyCreateFailed` | 502 | Bunny video create request failed |
| 9013 | `BunnyUploadFailed` | 502 | Bunny video upload request failed |
| 9014 | `BunnyInvalidAPIResponse` | 502 | Bunny returned invalid response |
| 9015 | `BunnyVideoNotFound` | 502 | Bunny video not found |
| 9016 | `BunnyGetVideoFailed` | 502 | Bunny GET video request failed |
| 9017 | `ImageEncodeBusy` | 503 | WebP encode gate at capacity |

---

## Implementation Reference

| Concern | Location |
|---------|----------|
| File entity + `MediaGateway` port | `internal/media/domain/` |
| MediaService use-cases | `internal/media/application/service.go` |
| `MediaGateway` implementation | `internal/media/infra/storage_gateway.go` |
| GORM repositories | `internal/media/infra/repos.go` |
| Cloud SDK client init | `internal/media/infra/provider_runtime.go` (`main.go` → `mediainfra.Setup()`) |
| Classification + list filters | `internal/media/infra/media_classification.go` |
| Storage MIME canonicalization | `internal/media/infra/media_storage_mime.go` |
| Metadata + upload entity | `internal/media/infra/media_entity_metadata.go` |
| WebP encoding | `internal/shared/utils/webp_encode.go` (CGO), `webp_encode_stub.go` (no-CGO) |
| HTTP handlers | `internal/media/delivery/handler.go`, `webhook_handler.go` |
| Route registration | `internal/media/delivery/routes.go` |
| Orphan cleanup jobs | `internal/media/jobs/cleanup_job.go`, `enqueue.go` |
| DB migrations | `migrations/000003_media_metadata.*`, `000004_media_orphan_safety.*`, `000005_media_bunny_response_fields.*`, `000008_media_metadata_json_storage.*` |
