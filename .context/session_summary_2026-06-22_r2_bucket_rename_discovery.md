# Discovery — Rename B2Bucket → R2Bucket (2026-06-22)

> Phase đầu — trước khi sửa code

## Scope

| Old | New |
|-----|-----|
| `MediaUploadEntityInput.B2Bucket` | `R2Bucket` |
| `File.B2BucketName` | `R2BucketName` |
| DB column `b2_bucket_name` | `r2_bucket_name` (migration 000025) |
| JSON `b2_bucket_name` | `r2_bucket_name` |

## GitNexus

- `B2BucketName` impact: LOW, 0 d=1 in graph
- Callers: repos, service, dto, mapping, media_upload_entity, upsert_columns_test

## FE

- No `b2_bucket_name` usage in fe-mycourse types

## Checklist đầu giai đoạn ✅
