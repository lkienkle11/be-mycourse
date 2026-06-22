# Session summary — R2 migration DISCOVERY (2026-06-22)

> **Phase:** Đầu giai đoạn (be-mycourse checklist) — **CHƯA SỬA CODE**

## Task

Migrate non-video media/file upload từ Backblaze B2 + Gcore CDN sang Cloudflare R2 (S3 API + custom domain public URL). Video giữ Bunny Stream. Test: login + upload 5 ảnh từ `temporary-docs/cac_file_media_cu`.

## 1. Context đã đọc

| File | Ghi chú |
|------|---------|
| `session_summary_2026-05-27_media_delete_provider_routing.md` | Delete routing dùng persisted `provider` row; Bunny GUID priority |
| `docs/modules/media.md` | Hiện ghi B2+Gcore; DTO đã có filter `R2` nhưng chưa implement |
| `temporary-docs/tai-lieu-cloudflare-r2-setup/ChatGPT-Cấu hình Cloudflare R2.md` | R2 env, S3 SDK, URL = `R2_PUBLIC_URL/key` (không bucket segment) |

## 2. Git baseline

- Branch: `chore/pm2-dynamic-deploy-paths` (ahead 1 vs origin)
- HEAD: `b6eb9bc` — không có diff local uncommitted liên quan R2
- Large diff vs `master` là refactor DDD trước đó (639 files) — task R2 là additive trên `internal/media/infra`

## 3. GitNexus research

- Index: **was stale (16 commits)** → refreshed `npx gitnexus analyze --force` ✅
- Subagent research: [2bb42a9d-259f-491e-9b51-45263f293903]

### Impact (upstream)

| Symbol | Risk | d=1 callers |
|--------|------|-------------|
| `NewCloudClientsFromSetting` | **CRITICAL** | `wireCore`, `infra.Setup` |
| `UploadB2` | LOW | `StorageGateway.UploadToProvider` |
| `BuildPublicURL` (clients.go) | LOW | `GetFile`, tests |
| `ResolveMediaProvider` | LOW | `ResolveUploadProvider`, `DefaultMediaProvider` |

**CRITICAL warning acknowledged:** `NewCloudClientsFromSetting` phải giữ startup OK khi R2 creds thiếu (graceful skip) hoặc fail rõ ràng — không break `main` nếu chỉ thiếu B2 cũ.

## 4. Deliverable — symbols

### REUSE

- `StorageGateway`, `MediaGateway`, upload helpers, `BuildB2ObjectKey` (key format), Bunny path, `FileProviderR2` constant, orphan/cleanup job pattern

### CHANGE

- `NewCloudClientsFromSetting` → attach R2 S3 client (aws-sdk-go-v2)
- `UploadB2` default branch → `UploadR2`; giữ `UploadB2`/`DeleteB2Object` cho legacy rows
- `BuildPublicURL` → R2 branch: `R2PublicURL + key`
- `ResolveMediaProvider` → default non-video = `R2`
- `ParseImageURLForOrphanCleanup` → match R2 public URL + legacy B2/Gcore
- `setting.Media` + YAML + `.env*` → `MEDIA_R2_*`
- `service_upload_helpers` / `service.go` → bucket name từ `R2Bucket`

### Docs gap (sync sau implement)

- `docs/modules/media.md`, `architecture.md`, `dependencies.md`, `data-flow.md`, `reusable-assets.md`, `database.md`, `README.md`, `.env*.example`, `IMPLEMENTATION_PLAN_EXECUTION.md`

## 5. Implementation plan (giai đoạn code)

1. Add R2 settings + config YAML/env
2. Add `clients_r2.go` (UploadR2, DeleteR2Object, URL builder)
3. Wire `CloudClients.R2Client`, update gateway routing
4. Update resolver, orphan URL parser, tests
5. Sync docs
6. Quality gates + manual upload test

## Checklist đầu giai đoạn

- [x] Context + docs đã đọc; docs gap liệt kê
- [x] git log + status reviewed
- [x] GitNexus query + context + impact + analyze refresh
- [x] Research note (reuse/callers/risk)
- [x] Source baseline khớp graph
- [x] **Chưa** sửa code / migration / docs (trước file này)
