# Media Module

## Overview

The media module (`internal/media/`) provides a unified API surface for file and video uploads with cloud storage providers. It handles:

- **Non-video files** — upload to Backblaze B2; public URL via Gcore CDN
- **Videos** — upload and stream via BunnyCDN Stream
- **Local** — reversible signed URL token (dev/fallback)

Provider selection is **server-side only** (`setting.MediaSetting.AppMediaProvider`) — never accepted from client request params.

---

## Directory Layout

```
internal/media/
├── domain/
│   ├── file.go                    # File entity
│   ├── repository.go              # FileRepository + PendingCleanupRepository interfaces
│   ├── errors.go                  # Domain errors
│   ├── bunny_status_codes.go      # Bunny numeric video status constants
│   ├── bunny_webhook.go           # Bunny webhook payload types
│   └── meta_keys.go               # JSON metadata key constants (video_id, thumbnail_url, etc.)
├── application/
│   ├── media_service.go           # MediaService: create/update/delete/list files, batch delete, video status, webhook
│   └── service_upload_helpers.go  # Upload orchestration helpers
├── infra/
│   ├── gorm_file_repo.go          # GormFileRepository
│   ├── gorm_cleanup_repo.go       # GormPendingCleanupRepository
│   ├── cloud_clients.go           # B2 + BunnyCDN SDK client init (NewCloudClientsFromSetting)
│   ├── media_metadata.go          # Typed metadata inference (image/video/document)
│   ├── webp_encode.go             # WebP encoding via bimg/libvips (CGO)
│   ├── webp_encode_stub.go        # Stub for CGO_ENABLED=0 builds
│   └── bunny_webhook_*.go         # Bunny webhook signature verification
├── delivery/
│   ├── handler.go                 # HTTP handlers
│   ├── routes.go                  # RegisterRoutes (authenticated) + RegisterWebhookRoutes (no-auth)
│   ├── dto.go                     # UploadFileResponse, VideoStatusResponse, etc.
│   ├── mapping.go                 # Domain/entity → DTO mapping
│   └── server_owned_test.go       # delivery_test
└── jobs/
    ├── orphan_enqueuer.go         # OrphanEnqueuer: enqueue superseded file cleanup
    ├── cleanup_scheduler.go       # Background cleanup worker
    ├── cleanup_constants.go       # Cleanup job constants
    └── global_counters.go         # GlobalCounters (metrics)
```

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

---

## Permissions

| Permission ID | Permission Name |
|--------------|----------------|
| P26 | `media_file:read` |
| P27 | `media_file:create` |
| P28 | `media_file:update` |
| P29 | `media_file:delete` |

---

## Upload Pipeline

### Single file upload (`POST /media/files`)

1. Bind multipart form fields (1–5 `files` parts; legacy `file` field accepted for single part).
2. Validate per-part size ≤ `MaxMediaUploadFileBytes` (2 GiB).
3. Validate aggregate size ≤ `MaxMediaMultipartTotalBytes` (2 GiB).
4. Validate part count ≤ `MaxMediaFilesPerRequest` (5).
5. For each part concurrently (up to `MaxConcurrentMediaUploadWorkers = 5`):
   a. **Executable check** (non-image, non-video only): reject if extension or magic bytes match denylist → HTTP 400 / code 2004.
   b. **WebP encoding** (images only): acquire encode gate, run `EncodeWebP` via bimg/libvips (CGO), update payload/filename/MIME/size.
   c. **Provider upload**: B2 (non-video) or Bunny Stream (video).
   d. **Metadata inference**: typed `ImageMetadata`, `VideoMetadata`, or `DocumentMetadata` inferred server-side from MIME/extension.
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

- Implementation: `bimg`/libvips (CGO). Requires `CGO_ENABLED=1` and `libvips-dev pkg-config` at build time.
- Concurrency: bounded by a semaphore in `internal/shared/utils/` (`AcquireEncodeGate` / `ReleaseEncodeGate`); cap = `MaxConcurrentImageEncode` (4).
- `CGO_ENABLED=0` builds: stub returns `ErrImageEncodeBusy` → HTTP 503 / code 9017.
- After encoding: `payload`, `filename` (`.webp`), `mime` (`image/webp`), and `sizeBytes` are updated before provider upload.

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
| `length`                       | `duration`           | Seconds (int)                          |
| `video_provider`               | `video_provider`     | `bunny_stream`                         |
| `b2_bucket_name`               | `b2_bucket_name`     | Set on B2 uploads                      |

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
3. `{cdn_hostname}/{guid}/{thumbnailFileName}` (the canonical case today)

### Bunny webhook (`POST /webhook/bunny`)

Mounted on the **no-filter lane** (no JWT, no rate limit).

1. Validate Bunny signature v1 on raw request body.
   - Headers required: `X-BunnyStream-Signature-Version: v1`, `X-BunnyStream-Signature-Algorithm: hmac-sha256`, `X-BunnyStream-Signature: <hex>`
   - Signature: `hex(HMAC-SHA256(rawBody, signingSecret))`
   - Signing secret: `setting.MediaSetting.BunnyStreamReadOnlyAPIKey` (fallback: `BunnyStreamAPIKey`)
2. Parse and validate JSON payload (`VideoLibraryId`, `VideoGuid`, `Status` 0..10).
3. Handle by status:
   - `Finished (3)` and `Resolution finished (4)`: load row, fetch Bunny detail,
     **merge metadata additively** (never overwrite valid prior data with zero
     values), update DB. The `thumbnail_url` and `duration` columns are only
     overwritten when the new value is non-empty / positive.
   - `Failed (5)` and `PresignedUploadFailed (8)`: mark row `FAILED` (idempotent).
   - Other statuses: accepted and intentionally ignored (idempotent callbacks).

---

## Response Contract (`UploadFileResponse`)

Public fields returned in API responses:

| Field | Notes |
|-------|-------|
| `url` | Public distribution URL |
| `object_key` | Storage object key |
| `metadata` | **Typed** sub-object (`UploadFileMetadata`). Normalised shape for image / video / document. See "Typed `metadata` shape" below. **This is the only metadata sub-object exposed to clients.** |
| `bunny_video_id` | Bunny video GUID |
| `bunny_library_id` | Bunny library ID |
| `video_id` | Numeric Bunny ID or GUID |
| `thumbnail_url` | Built from `BunnyStreamCDNHostname` + `guid` + `thumbnailFileName` (current Bunny API); falls back to `thumbnailUrl` / `defaultThumbnailUrl` if Bunny ever populates them |
| `embeded_html` | Escaped `<iframe ...>` embed HTML |
| `duration` | Video duration in seconds |
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
builder (`BuildTypedMetadata`) so the shape is always consistent:

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
remain possible. For Bunny videos this includes:

```jsonc
{
  "video_guid": "7312c208-054f-413f-82b2-3666a271f4f4",
  "bunny_video_id": "7312c208-054f-413f-82b2-3666a271f4f4",
  "bunny_library_id": "650694",
  "video_provider": "bunny_stream",
  "video_id": "7312c208-054f-413f-82b2-3666a271f4f4",
  "thumbnail_url": "https://vz-xxxxxxxx-xxxx.b-cdn.net/.../thumbnail.jpg",
  "embeded_html": "<iframe ...></iframe>",
  // Typed-mirrored keys (consumed by BuildTypedMetadata)
  "length": 190,
  "width": 1920,
  "height": 1080,
  "framerate": 23.976,
  "video_codec": "x264",
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

- **FK-based (Sub 14):** taxonomy `categories.image_file_id` or `users.avatar_file_id` replacement → `OrphanEnqueuer.EnqueueOrphanCleanupForFileID`
- **URL-based (Sub 07):** `EnqueueOrphanImageCleanupByURL` — resolves URL pattern to cloud object key

The background cleanup worker in `jobs/cleanup_scheduler.go` processes `media_pending_cloud_cleanup` rows asynchronously. Local provider rows are skipped.

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
| 9010 | `B2BucketNotConfigured` | 500 | B2 bucket not configured |
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
| File entity + repository interfaces | `internal/media/domain/` |
| MediaService use-cases | `internal/media/application/media_service.go` |
| GORM repositories | `internal/media/infra/gorm_file_repo.go`, `gorm_cleanup_repo.go` |
| Cloud SDK client init | `internal/media/infra/cloud_clients.go` |
| Metadata inference | `internal/media/infra/media_metadata.go` |
| WebP encoding | `internal/shared/utils/webp_encode.go` (CGO), `webp_encode_stub.go` (no-CGO) |
| HTTP handlers | `internal/media/delivery/handler.go` |
| Route registration | `internal/media/delivery/routes.go` |
| Orphan cleanup jobs | `internal/media/jobs/` |
| DB migrations | `migrations/000003_media_metadata.*`, `000004_media_orphan_safety.*`, `000005_media_bunny_response_fields.*` |
