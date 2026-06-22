# Closeout — Rename B2Bucket → R2Bucket (2026-06-22)

## Phase 3 checklist

| Step | Status |
|------|--------|
| `make test-all` | PASS |
| Docs audit (`rg b2_bucket` in code) | PASS — chỉ còn migration 000003 (historical), 000025, `.context/` |
| Migration 000025 | CREATED |
| GitNexus detect_changes | run at closeout |
| Manual upload test | BLOCKED — R2 creds empty in `.env` |

## Changes summary

### Go identifiers
| Before | After |
|--------|-------|
| `MediaUploadEntityInput.B2Bucket` | `R2Bucket` |
| `File.B2BucketName` | `R2BucketName` |
| `storageBucketFromUploadInput` | `r2BucketFromUploadInput` |

### DB / API
| Before | After |
|--------|-------|
| column `b2_bucket_name` | `r2_bucket_name` (migration `000025`) |
| JSON `b2_bucket_name` | `r2_bucket_name` |
| raw metadata key `b2_bucket_name` | `r2_bucket_name` |

### Files touched
- `internal/media/domain/media.go`
- `internal/media/infra/repos.go`, `media_upload_entity.go`, `clients_r2.go`
- `internal/media/application/service.go`, `service_upload_helpers.go`
- `internal/media/delivery/dto.go`, `mapping.go`
- `internal/media/infra/upsert_columns_test.go`
- `docs/api_swagger.yaml`, `docs/database.md`, `docs/modules/media.md`
- `migrations/000025_media_r2_bucket_name.{up,down}.sql`

## Breaking change note

API response field đổi từ `b2_bucket_name` → `r2_bucket_name`. FE hiện không dùng field này; nếu có client cũ cần cập nhật.

## Deploy note

Chạy migration trước khi deploy code mới:
```sql
ALTER TABLE media_files RENAME COLUMN b2_bucket_name TO r2_bucket_name;
```
