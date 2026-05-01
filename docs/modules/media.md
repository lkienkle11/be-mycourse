# Media Module


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

> **Status:** Sub 04 (B2 URL, keys, Bunny status + webhook), Sub 06 (`row_version`, `content_fingerprint`, deferred cleanup), Sub 09 (Bunny parity: `video_id`, `thumbnail_url`, `embeded_html` on API + `media_files` + metadata JSON).

**Cross-references:** `docs/return_types.md` (JSON examples), `docs/api_swagger.yaml` (`UploadFileResponse`), `docs/data-flow.md`, `docs/reusable-assets.md`, `docs/database.md` (`media_files`), `migrations/README.md` (000003–000005), `IMPLEMENTATION_PLAN_EXECUTION.md` (execution notes).

---

## Overview

Media module provides a unified API surface for file and video uploads with provider-aware URL behavior:

- Non-video files: upload storage at B2, distribution URL through Gcore CDN as **`{CDN}/{bucket}/{object_key}`** (bucket from `setting.MediaSetting.B2Bucket` after `setting.Setup()`, falling back to the env bucket used when constructing the B2 client). Object keys for B2 default uploads: **8 random digits + `-` + sanitized filename** (`pkg/logic/helper/media_upload_keys.go`). Bunny Stream videos use the API **GUID** as `object_key`.
- Videos: playback/distribution URL through Bunny Stream.
- Local provider: reversible signed URL token that can be decoded back to object key.

Media persists upload metadata into `media_files` after successful cloud operations (create/update) and marks rows deleted on successful delete sync. **Phase Sub 06** adds optimistic concurrency (`row_version`), SHA-256 hex `content_fingerprint`, deferred superseded-cloud deletes (`media_pending_cloud_cleanup` + worker in `internal/jobs/media_pending_cleanup_scheduler.go`), and multipart field binding via `pkg/logic/helper/media_multipart.go` (aligned with `ParseMetadataFromRaw`).

SDK clients are initialized once at app startup (`pkg/media.Setup()` in `main.go`), then reused by media service flow.
Provider source-of-truth is server-side config (`setting.MediaSetting.AppMediaProvider`) and is never accepted from client request params.

**Layering:** Kind/provider resolution and **Bunny presentation policy** (thumbnail URL fields from GET-video JSON, `video_id` string from numeric `id` vs `guid`, embed URL + iframe HTML in metadata) live in **`pkg/logic/helper/media_resolver.go`** (`EnrichBunnyVideoDetail`, `ApplyBunnyDetailToMetadata`, `ResolveBunnyEmbedURL`, …). **`pkg/media/clients.go`** performs Bunny HTTP and `json.Unmarshal` into `pkg/entities.BunnyVideoDetail`, then invokes those helpers. **`services/media`** orchestrates only.

Metadata parsing and typed inference are handled in helper layer (`pkg/logic/helper/media_metadata.go`) instead of service layer.
`kind` and `metadata` values coming from multipart request text fields are parsed only for backward-compat validation and then ignored; server-side extractor/provider outputs are the only source for persisted and returned metadata.
When server cannot infer kind from MIME/extension and no app-level provider override is configured, upload provider falls back to `Local` by policy.
Generic raw metadata primitives (`DetectExtension`, `ImageSizeFromPayload`, `StringFromRaw`, `IntFromRaw`, `FloatFromRaw`, `NonEmpty`), multipart loose-bool parsing (`ParseBoolLoose`), and upload-byte fingerprinting (`ContentFingerprint`) live in `pkg/logic/utils/parsing.go` and must be called through `utils.*` import alias in helper/service code.
Public API responses are mapped by `pkg/logic/mapping` to `dto.UploadFileResponse`, and internal provider selection is **not** exposed as a top-level field.

---

## API Surface

| Method | Path | Purpose |
|---|---|---|
| OPTIONS | `/media/files` | CORS/preflight support |
| GET | `/media/files` | List persisted records from `media_files` with pagination |
| GET | `/media/files/cleanup-metrics` | Ops: deferred cleanup counters — registered **before** `/:id` |
| POST | `/media/files` | Upload multipart file and return file descriptor |
| OPTIONS | `/media/files/:id` | CORS/preflight support |
| GET | `/media/files/:id` | Build file detail from object key |
| PUT | `/media/files/:id` | Re-upload/replace by logical row (`id` = `object_key`). Optional: `reuse_media_id`, `expected_row_version`, `skip_upload_if_unchanged`. Conflict → **409**. |
| DELETE | `/media/files/:id` | Delete object on configured provider |
| OPTIONS | `/media/files/local/:token` | CORS/preflight support |
| GET | `/media/files/local/:token` | Decode local signed token to object key |
| GET | `/media/videos/:id/status` | Bunny video processing status by GUID |
| POST | `/webhook/bunny` | Bunny callback (no auth middleware) |

---

## Permissions

| Permission ID | Permission Name |
|---|---|
| P26 | `media_file:read` |
| P27 | `media_file:create` |
| P28 | `media_file:update` |
| P29 | `media_file:delete` |

Role mapping: `constants/roles_permission.go` — sync via `go run ./cmd/syncpermissions` and `go run ./cmd/syncrolepermissions`.

---

## Runtime descriptor (`dto.UploadFileResponse`)

- `url`, `origin_url`, `object_key`
- `metadata`: typed **`UploadFileMetadata`** (nested), zero defaults when unknown
- **Not returned:** internal `provider` string (server-owned routing)

**Top-level Bunny-related fields (Sub 09; JSON `omitempty` when empty):**

| Field | Meaning |
|-------|---------|
| `bunny_video_id` | Bunny video GUID |
| `bunny_library_id` | Stream library id |
| `video_id` | Numeric Bunny **`id`** from GET-video when unmarshalled; else GUID from `FormatBunnyVideoIDString` |
| `thumbnail_url` | From API `thumbnailUrl` / `defaultThumbnailUrl` after `EnrichBunnyVideoDetail` |
| `embeded_html` | Escaped `<iframe …>`; `ResolveBunnyEmbedHTML` from play-base URL → `/embed/{libraryId}/{guid}` |
| `duration`, `video_provider`, `row_version`, `content_fingerprint` | As before (Sub 06 for last two) |

**Persistence:** Same logical values appear in **`media_files`** columns (`video_id`, `thumbnail_url`, `embeded_html`, migration **`000005_media_bunny_response_fields`**) and in **`metadata_json`** under keys from **`constants/media_meta_keys.go`**. `ToMediaEntity` prefers columns, then JSON fallback for legacy rows.

**Upload enrichment:** After Bunny PUT, `UploadBunnyVideo` calls **`GetBunnyVideoByID`**. **Only if that succeeds**, `ApplyBunnyDetailToMetadata` merges the three keys into provider metadata. If GET fails, those keys may stay empty until **`HandleBunnyVideoWebhook`** (finished) refreshes them.

**Webhook:** On finished status, webhook loads row, merges technical fields, calls **`ApplyBunnyDetailToMetadata`**, sets `row.VideoID` / `ThumbnailURL` / `EmbededHTML` from merged map, then `UpsertByObjectKey`.

---

## Phase Sub 06 — orphan / reuse / deferred cleanup (summary)

- **Migration:** `000004_media_orphan_safety.up.sql` — `row_version`, `content_fingerprint`; table `media_pending_cloud_cleanup`.
- **Superseded cloud object:** queued for deferred delete, not inline at replace time.
- **Reuse / fingerprint / optimistic lock:** see bullets in earlier sections; errors **409** with `ErrMediaOptimisticLock` / `ErrMediaReuseMismatch`.
- **Worker:** `internal/jobs/media_pending_cleanup_scheduler.go`; logic `services/media/pending_cleanup.go`.

---

## Upstream errors (Sub 04)

- B2 bucket not configured → **9010** (`B2BucketNotConfigured`), HTTP **500**.
- Bunny Stream not configured → **9011**.
- Bunny create / upload / invalid API response → **9012** / **9013** / **9014** (HTTP **502**).
- Bunny video not found / Bunny get-video failed → **9015** / **9016**.
- Messages: **`constants/error_msg.go`**; **`pkg/errcode/messages.go`** imports those constants.

---

## Upload size and transport

- **2 GiB** per `file` part: **`constants.MaxMediaUploadFileBytes`**, **`constants.MsgFileTooLargeUpload`** — HTTP **413**, code **2003**; missing `file` → **400**, **3001**.
- Gin **`MaxMultipartMemory`** 64 MiB — see `docs/router.md`, `docs/deploy.md` for proxy **`client_max_body_size`**.

---

## Provider rules

- `Local`, B2+Gcore, Bunny Stream — provider from server config only; `setting.MediaSetting` after `Setup()`.
- Local token decode: **`pkg/logic/helper/DecodeLocalURLToken`**.

---

## Testing

- All tests under repository root **`tests/`** (`README.md`, `docs/patterns.md`).
- Bunny / mapping: `tests/sub04_media_pipeline_test.go`; orphan safety: `tests/sub06_media_orphan_safety_test.go` where applicable.
