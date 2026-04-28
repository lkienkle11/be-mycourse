# Media Module


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

> **Status: Implemented through Phase Sub 04 (B2 URL path, keys, Bunny status + webhook).**

---

## Overview

Media module provides a unified API surface for file and video uploads with provider-aware URL behavior:

- Non-video files: upload storage at B2, distribution URL through Gcore CDN as **`{CDN}/{bucket}/{object_key}`** (bucket from `setting.MediaSetting.B2Bucket` after `setting.Setup()`, falling back to the env bucket used when constructing the B2 client). Object keys for B2 default uploads: **8 random digits + `-` + sanitized filename** (`pkg/logic/helper/media_upload_keys.go`). Bunny Stream videos use the API **GUID** as `object_key`.
- Videos: playback/distribution URL through Bunny Stream.
- Local provider: reversible signed URL token that can be decoded back to object key.

Media now persists upload metadata into `media_files` after successful cloud operations (create/update) and marks rows deleted on successful delete sync.

SDK clients are initialized once at app startup (`pkg/media.Setup()` in `main.go`), then reused by media service flow.
Provider source-of-truth is server-side config (`setting.MediaSetting.AppMediaProvider`) and is never accepted from client request params.
Media kind/provider normalization is implemented as shared helper assets in `pkg/logic/helper/media_resolver.go`, keeping `services/media` orchestration-only.
Metadata parsing and typed inference are handled in helper layer (`pkg/logic/helper/media_metadata.go`) instead of service layer.
Generic raw metadata primitives (`DetectExtension`, `ImageSizeFromPayload`, `StringFromRaw`, `IntFromRaw`, `FloatFromRaw`, `NonEmpty`) are extracted to `pkg/logic/utils/media_metadata.go` and must be called through `utils.*` import alias in helper code.
Public API responses are mapped by `pkg/logic/mapping` to `dto.UploadFileResponse`, and internal provider details are removed from public payload.

---

## API Surface

| Method | Path | Purpose |
|---|---|---|
| OPTIONS | `/media/files` | CORS/preflight support |
| GET | `/media/files` | List persisted records from `media_files` with pagination |
| POST | `/media/files` | Upload multipart file and return file descriptor |
| OPTIONS | `/media/files/:id` | CORS/preflight support |
| GET | `/media/files/:id` | Build file detail from object key |
| PUT | `/media/files/:id` | Re-upload/replace object by object key |
| DELETE | `/media/files/:id` | Delete object on configured provider |
| OPTIONS | `/media/files/local/:token` | CORS/preflight support |
| GET | `/media/files/local/:token` | Decode local signed token to object key |
| GET | `/media/videos/:id/status` | Get Bunny video processing status by GUID |
| POST | `/webhook/bunny` | Bunny callback endpoint (registered outside auth/permission middleware) |

---

## Permissions

| Permission ID | Permission Name |
|---|---|
| P26 | `media_file:read` |
| P27 | `media_file:create` |
| P28 | `media_file:update` |
| P29 | `media_file:delete` |

Role mapping is declared in `constants/roles_permission.go` and synced via:

- `go run ./cmd/syncpermissions`
- `go run ./cmd/syncrolepermissions`

---

## Runtime Descriptor

Returned `dto.UploadFileResponse` fields:

- `url`: effective URL returned to client
- `origin_url`: provider origin URL
- `object_key`: storage key/object identifier
- `provider` is intentionally **not returned** in public response to avoid exposing internal provider selection policy.
- `metadata`: typed object inferred by backend:
  - `ImageMetadata` for image files
  - `VideoMetadata` for video files
  - `DocumentMetadata` for non-image documents

`VideoMetadata` includes:
- `duration`
- `width`
- `height`
- `bitrate`
- `fps`
- `video_codec`
- `audio_codec`
- `has_audio`
- `is_hdr`

`UploadFileResponse` (top-level) includes:
- `bunny_video_id`
- `bunny_library_id`
- `duration`
- `video_provider`

### Upstream errors (Sub 04)

- B2 bucket missing at runtime (no resolved bucket after settings + env fallback) → application code **9010** (`B2BucketNotConfigured`), HTTP **500**.
- Bunny Stream not configured → **9011**.
- Bunny create / upload / invalid API response → **9012** / **9013** / **9014** (HTTP **502**).
- Bunny video not found / Bunny get-video failed → **9015** / **9016**.
- Default JSON messages for **9010–9016** live in **`constants/error_msg.go`**; **`pkg/errcode/messages.go`** references those constants only.

---

## Upload size and transport

- **Per-file cap:** each multipart part `file` on `POST` / `PUT` is limited to **2 GiB** (`2×1024×1024×1024` bytes). Limit and the **single** oversize message constant are in **`constants/error_msg.go`**: `MaxMediaUploadFileBytes`, **`MsgFileTooLargeUpload`** (used by both `pkg/errcode` default for `FileTooLarge` / JSON `message` and `pkg/errors.ErrFileExceedsMaxUploadSize` — see file header; do not duplicate the literal in `pkg/errcode/messages.go`). Oversize → HTTP **413 Request Entity Too Large** with envelope `code` **2003** (`FileTooLarge`). A missing `file` part stays **400** with `code` **3001** (`BadRequest`) and message `file is required (multipart field: file)` — different from oversize.
- **Oversize sentinel:** `pkg/errors.ErrFileExceedsMaxUploadSize` (`pkg/errors/upload_errors.go`) = `errors.New(constants.MsgFileTooLargeUpload)` (no ad-hoc string in `services/media`). Handler maps with `errors.Is` to `FileTooLarge`; API `message` comes from `errcode.DefaultMessage(FileTooLarge)` → same constant.
- **Gin:** `api.InitRouter` sets `MaxMultipartMemory` to **64 MiB** so multipart parsing does not keep entire large bodies in heap by default; larger parts are spilled to temporary disk files during parse (still subject to the 2 GiB application cap when reading the opened `multipart.File`).
- **Reverse proxy / load balancer:** configure body size limits **≥ 2 GiB** on the API route (e.g. nginx `client_max_body_size 2G;` on `api.*`) or the proxy rejects the request before it reaches Go. See `docs/deploy.md`.

---

## Provider Rules

- `Local`: `url` is a signed reversible token path (`/api/v1/media/files/local/:token`), `origin_url` stores raw object key.
- Non-video default: upload to B2 and return Gcore CDN URL + B2 origin URL.
- Video default: upload to Bunny Stream and return playback URL.
- Client request cannot override provider; provider is selected from server env config.
- Runtime provider/base URL/secret reads use `setting.MediaSetting` (post-`setting.Setup()` source of truth); direct `os.Getenv` is kept only in the approved env-only constructor path.
- Decode token flow uses helper placement (`pkg/logic/helper/DecodeLocalURLToken`), not service-local utility.

---

## Testing

- Add all tests for this API (unit/module-level/integration) under repository root **`tests/`** only (shared convention for the whole backend — see `README.md` **Testing** and `.full-project/patterns.md`).
- Latest verification (2026-04-27): `go build ./...` and `go test ./... -count=1` pass after fixing helper alias usage in `pkg/logic/helper/media_metadata.go` (`util.*` -> `utils.*`).

