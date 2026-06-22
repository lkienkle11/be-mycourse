# Session summary — Remove B2/Gcore DISCOVERY (2026-06-22)

> **Phase:** Đầu giai đoạn (be-mycourse checklist) — **CHƯA SỬA CODE** (file này)

## Task

Xóa **toàn bộ** code Backblaze B2 + Gcore CDN. Non-video upload chỉ còn Cloudflare R2. Video giữ Bunny Stream.

## GitNexus impact (trước khi sửa)

| Symbol | Risk | d=1 |
|--------|------|-----|
| `UploadB2` | LOW | `StorageGateway.UploadToProvider` |
| `DeleteB2Object` | LOW | `DeleteStoredObject` |
| `NewCloudClientsFromSetting` | HIGH | `infra.Setup`, `wireCore` |
| `attachB2FromSetting` | — | removed with B2 purge |

## Deliverable

### REUSE
- `UploadR2`, `DeleteR2Object`, `clients_r2.go`, Bunny path, `BuildB2ObjectKey` format → rename `BuildObjectStorageKey`

### REMOVE
- `UploadB2`, `DeleteB2Object`, `effectiveB2Bucket`, `b2UploadResultURLs`, `attachB2FromSetting`
- `B2Client`, `B2BucketName` from `CloudClients`
- `MediaSetting` B2/Gcore fields + YAML/env
- `orphanCleanupB2Match`, Gcore URL parsing
- deps: `github.com/Backblaze/blazer`, `github.com/G-Core/gcorelabscdn-go`
- errcode `9010` B2BucketNotConfigured

### CHANGE
- `UploadToProvider` / `DeleteStoredObject`: R2 only for non-video (legacy `provider=B2` rows → R2 delete)
- `BuildPublicURL`: remove B2 branch; R2 + Bunny + Local
- Tests + docs sync

## Docs gap
- `docs/modules/media.md`, `dependencies.md`, `architecture.md`, `data-flow.md`, `reusable-assets.md`, `README.md`, `.env*.example`, `config/app*.yaml`

## Checklist đầu giai đoạn
- [x] Context + docs gap
- [x] git reviewed
- [x] GitNexus query + impact
- [x] Research note
- [x] Chưa sửa code (trước implementation batch)
