# Media Module

> **Status: Implemented (Phase Sub 02).**

---

## Overview

Media module provides a unified API surface for file and video uploads with provider-aware URL behavior:

- Non-video files: upload storage at B2, distribution URL through Gcore CDN.
- Videos: playback/distribution URL through Bunny Stream.
- Local provider: reversible signed URL token that can be decoded back to object key.

Media does **not** persist file records in local database. Backend acts as upload gateway to third-party services.

SDK clients are initialized once at app startup (`pkg/media.Setup()` in `main.go`), then reused by media service flow.
Media kind/provider normalization is implemented as shared helper assets in `pkg/logic/helper/media_resolver.go`, keeping `services/media` orchestration-only.
Metadata raw parsing/normalization is also handled in helper layer (`pkg/logic/helper/media_metadata.go`) instead of service layer.

---

## API Surface (`/api/v1/media/files`)

| Method | Path | Purpose |
|---|---|---|
| OPTIONS | `/media/files` | CORS/preflight support |
| GET | `/media/files` | List endpoint (stateless placeholder) |
| POST | `/media/files` | Upload multipart file and return file descriptor |
| OPTIONS | `/media/files/:id` | CORS/preflight support |
| GET | `/media/files/:id` | Build file detail from object key + query params |
| PUT | `/media/files/:id` | Re-upload/replace object by object key |
| DELETE | `/media/files/:id` | Delete object on cloud provider |
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

Returned `File` object fields:

- `kind`: `FILE | VIDEO`
- `provider`: `S3 | GCS | B2 | R2 | Bunny | Local`
- `url`: effective URL returned to client
- `origin_url`: provider origin URL
- `object_key`: storage key/object identifier
- `status`: `READY | FAILED | DELETED`
- `metadata`: dynamic JSON map from request/provider response

---

## Provider Rules

- `Local`: `url` is a signed reversible token path (`/api/v1/media/files/local/:token`), `origin_url` stores raw object key.
- Non-video default: upload to B2 and return Gcore CDN URL + B2 origin URL.
- Video default: upload to Bunny Stream and return playback URL.

