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

After Bunny PUT, `GetBunnyVideoByID` is called. If successful, `ApplyBunnyDetailToMetadata` merges:
- `video_id`, `thumbnail_url`, `embeded_html`
- Video telemetry: `width`, `height`, `length`, `framerate`, `bitrate`, `video_codec`, `audio_codec`

Because Bunny may not have finished transcoding, `width`/`height` may be 0 immediately after upload. The Bunny webhook populates them when transcoding completes.

### Bunny webhook (`POST /webhook/bunny`)

Mounted on the **no-filter lane** (no JWT, no rate limit).

1. Validate Bunny signature v1 on raw request body.
   - Headers required: `X-BunnyStream-Signature-Version: v1`, `X-BunnyStream-Signature-Algorithm: hmac-sha256`, `X-BunnyStream-Signature: <hex>`
   - Signature: `hex(HMAC-SHA256(rawBody, signingSecret))`
   - Signing secret: `setting.MediaSetting.BunnyStreamReadOnlyAPIKey` (fallback: `BunnyStreamAPIKey`)
2. Parse and validate JSON payload (`VideoLibraryId`, `VideoGuid`, `Status` 0..10).
3. Handle by status:
   - `Finished (3)` and `Resolution finished (4)`: load row, fetch Bunny detail, merge metadata, update DB.
   - `Failed (5)` and `PresignedUploadFailed (8)`: mark row `FAILED` (idempotent).
   - Other statuses: accepted and intentionally ignored (idempotent callbacks).

---

## Response Contract (`UploadFileResponse`)

Public fields returned in API responses:

| Field | Notes |
|-------|-------|
| `url` | Public distribution URL |
| `object_key` | Storage object key |
| `metadata` | Typed `UploadFileMetadata` (nested) |
| `bunny_video_id` | Bunny video GUID |
| `bunny_library_id` | Bunny library ID |
| `video_id` | Numeric Bunny ID or GUID |
| `thumbnail_url` | From Bunny `thumbnailUrl` / `defaultThumbnailUrl` |
| `embeded_html` | Escaped `<iframe ...>` embed HTML |
| `duration` | Video duration in seconds |
| `row_version` | Optimistic concurrency version (Sub 06) |
| `content_fingerprint` | SHA-256 hex of file bytes (Sub 06) |

**`origin_url` is NOT in the public API response** (Sub 12). The DB column and `File.OriginURL` (JSON `"-"`) are server-only for orphan resolution and delete routing.

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
