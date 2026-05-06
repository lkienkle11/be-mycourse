# PostgreSQL migrations

Các file `*_up.sql` được **nhúng (embed)** vào binary (`migrations/embed.go`) và chạy theo **số version** ở đầu tên file (`000001`, `000002`, …). Bảng `schema_migrations` trên Postgres ghi version đã áp dụng.

## Chuỗi hiện tại

| File | Nội dung |
|------|----------|
| `000001_schema` | RBAC + `users` (kể cả `refresh_token_session`), seed quyền/role. |
| `000002_taxonomy_domain` | Taxonomy: `course_levels`, `categories`, `tags`, … |
| `000003_media_metadata` | Bảng **`media_files`** (upload gateway), index. |
| `000004_media_orphan_safety` | Cột `media_files.row_version`, `content_fingerprint`; bảng **`media_pending_cloud_cleanup`**. |
| `000005_media_bunny_response_fields` | Cột **`media_files.video_id`**, **`thumbnail_url`**, **`embeded_html`** (Bunny parity / API `UploadFileResponse` — xem `docs/modules/media.md`). |
| `000006_taxonomy_user_media_refs` | `categories.image_file_id` + `users.avatar_file_id` (FK → `media_files.id`); drop cột URL thuần `image_url` / `avatar_url`; backfill FK từ URL khớp `media_files.url` / `origin_url`. |

**Xóa toàn bộ bảng bằng SQL (đúng thứ tự FK):** xem `docs/database.md` — mục **Drop All Tables**; khi thêm bảng mới cập nhật danh sách `DROP TABLE` tương ứng.

**Quy ước:** `permission_id` dạng `P{number}`; `permission_name` dạng `resource:action` (JWT / `RequirePermission`). Khi thêm quyền mới: cập nhật `constants/permissions.go`, migration nếu cần seed, rồi `go run ./cmd/syncpermissions` trên môi trường đã có dữ liệu. Ma trận role: `constants/roles_permission.go` + `go run ./cmd/syncrolepermissions`.

## Cách chạy migration với Gin / server hiện tại

1. Cấu hình Postgres giống lúc chạy app (`config/app.yaml` + `.env` — mục `[database]`).
2. Trong **PowerShell**:

```powershell
$env:MIGRATE = "1"
go run .
```

Hoặc build rồi chạy:

```powershell
$env:MIGRATE = "1"
.\mycourse-io-be.exe
```

3. App sẽ migrate rồi khởi động Gin. Muốn chỉ migrate rồi thoát, có thể `Ctrl+C` sau log thành công.

## Thêm migration mới

1. Tạo cặp `00000N_mô_tả.up.sql` và `00000N_mô_tả.down.sql` (tăng số so với bản mới nhất).
2. **golang-migrate** tách lệnh theo `;` và **không** bỏ qua comment `--` — **không đặt `;` trong cùng dòng `-- ...`** (sẽ cắt nhầm). Tránh `;` trong chuỗi hoặc `$$ ... $$` nếu không chủ ý.
3. `go build` để embed file mới.
4. Cập nhật bảng trên trong **`migrations/README.md`** và **`docs/database.md`** (Migration history + bảng chi tiết nếu cần).

## Rollback (down)

App **không** tự chạy `.down.sql`. Rollback: chạy tay SQL down theo thứ tự ngược version, hoặc chỉnh `schema_migrations` khi hiểu rủi ro.

## RBAC phẳng

Không có hierarchy role. Quyền hiệu lực = hợp quyền từ mọi role của user + `user_permissions`. Sau khi user xác nhận email, app gán role `learner` (cần đã chạy `000001`).
