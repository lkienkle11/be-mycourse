# Media Module

> **Status: Implemented through Phase Sub 04 (B2 URL path, keys, Bunny errors).**

---

## Overview

Media module provides a unified API surface for file and video uploads with provider-aware URL behavior:

- Non-video files: upload storage at B2, distribution URL through Gcore CDN as **`{CDN}/{bucket}/{object_key}`** (bucket from `setting.MediaSetting.B2Bucket` after `setting.Setup()`, falling back to the env bucket used when constructing the B2 client). Object keys for B2 default uploads: **8 random digits + `-` + sanitized filename** (`pkg/logic/helper/media_upload_keys.go`). Bunny Stream videos use the API **GUID** as `object_key`.
- Videos: playback/distribution URL through Bunny Stream.
- Local provider: reversible signed URL token that can be decoded back to object key.

Media does **not** persist file records in local database. Backend acts as upload gateway to third-party services.

SDK clients are initialized once at app startup (`pkg/media.Setup()` in `main.go`), then reused by media service flow.
Provider source-of-truth is server-side config (`setting.MediaSetting.AppMediaProvider`) and is never accepted from client request params.
Media kind/provider normalization is implemented as shared helper assets in `pkg/logic/helper/media_resolver.go`, keeping `services/media` orchestration-only.
Metadata parsing and typed inference are handled in helper layer (`pkg/logic/helper/media_metadata.go`) instead of service layer.
Generic raw metadata primitives (`DetectExtension`, `ImageSizeFromPayload`, `StringFromRaw`, `IntFromRaw`, `FloatFromRaw`, `NonEmpty`) are extracted to `pkg/logic/util/media_metadata.go`.
Public API responses are mapped by `pkg/logic/mapping` to `dto.UploadFileResponse`, and internal provider details are removed from public payload.

---

## API Surface (`/api/v1/media/files`)

| Method | Path | Purpose |
|---|---|---|
| OPTIONS | `/media/files` | CORS/preflight support |
| GET | `/media/files` | List endpoint (stateless placeholder) |
| POST | `/media/files` | Upload multipart file and return file descriptor |
| OPTIONS | `/media/files/:id` | CORS/preflight support |
| GET | `/media/files/:id` | Build file detail from object key |
| PUT | `/media/files/:id` | Re-upload/replace object by object key |
| DELETE | `/media/files/:id` | Delete object on configured provider |
| OPTIONS | `/media/files/local/:token` | CORS/preflight support |
| GET | `/media/files/local/:token` | Decode local signed token to object key |

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
- `metadata`: typed object inferred by backend:
  - `ImageMetadata` for image files
  - `VideoMetadata` for video files
  - `DocumentMetadata` for non-image documents

`VideoMetadata` always includes:
- `duration`
- `thumbnail_url`
- `bunny_video_id`
- `bunny_library_id`
- `video_provider` (e.g. `bunny_stream` set by `pkg/media` after upload)
- `size`
- `width`
- `height`

### Upstream errors (Sub 04)

- B2 bucket missing at runtime (no resolved bucket after settings + env fallback) → application code **9010** (`B2BucketNotConfigured`), HTTP **500**.
- Bunny Stream not configured → **9011**; create / upload / invalid API response → **9012** / **9013** / **9014** with HTTP **502** for **9012–9014** (see `pkg/media/provider_error.go` and `api/v1/media/file_handler.go`).
- Default JSON messages for **9010–9014** live in **`constants/error_msg.go`**; **`pkg/errcode/messages.go`** references those constants only.

---

## Upload size and transport

- **Per-file cap:** each multipart part `file` on `POST` / `PUT` is limited to **2 GiB** (`2×1024×1024×1024` bytes). Limit and the **single** oversize message constant are in **`constants/error_msg.go`**: `MaxMediaUploadFileBytes`, **`MsgFileTooLargeUpload`** (used by both `pkg/errcode` default for `FileTooLarge` / JSON `message` and `pkg/media.ErrFileExceedsMaxUploadSize` — see file header; do not duplicate the literal in `pkg/errcode/messages.go`). Oversize → HTTP **413 Request Entity Too Large** with envelope `code` **2003** (`FileTooLarge`). A missing `file` part stays **400** with `code` **3001** (`BadRequest`) and message `file is required (multipart field: file)` — different from oversize.
- **Oversize sentinel:** `pkg/media.ErrFileExceedsMaxUploadSize` (`pkg/media/upload_errors.go`) = `errors.New(constants.MsgFileTooLargeUpload)` (no ad-hoc string in `services/media`). Handler maps with `errors.Is` to `FileTooLarge`; API `message` comes from `errcode.DefaultMessage(FileTooLarge)` → same constant.
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

- Add **module-level / integration** tests for this API under repository root **`tests/`** (shared convention for the whole backend — see `README.md` **Testing** and `.full-project/patterns.md`). Narrow unit tests may still use colocated `*_test.go` next to `services/media` or `pkg/logic/*` when scoped to one package.

