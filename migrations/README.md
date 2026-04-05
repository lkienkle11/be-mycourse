# PostgreSQL migrations

Các file `*_up.sql` được **nhúng (embed)** vào binary (`migrations/embed.go`) và chạy theo **số version** ở đầu tên file (`000001`, `000002`, …). Bảng `schema_migrations` trên Postgres ghi version đã áp dụng.

## Chuỗi hiện tại (đã gộp)

| File | Nội dung |
|------|----------|
| `000001_schema` | Một lần tạo `permissions` (kèm `code_check`), `roles`, `role_permissions`, `users` (kèm `refresh_token_session` JSONB), `user_roles`, `user_permissions` với `user_id BIGINT` FK tới `users(id)`. |
| `000002_rbac_seed` | Toàn bộ catalog permission theo `constants/permissions.go`, bốn role `constants/roles.go`, ma trận `role_permissions` (admin = full catalog). Idempotent (`ON CONFLICT`). |

Các bước migrate cũ (`000003`–`000013`) đã bỏ. **Nếu DB từng chạy bản cũ** (version > 2 trong `schema_migrations`), cần **reset** DB hoặc xử lý tay `schema_migrations` — không hỗ trợ nâng cấp từ chuỗi cũ sang chuỗi mới.

**Xóa toàn bộ bảng bằng SQL (đúng thứ tự FK):** xem `docs/database.md` — mục **Xóa toàn bộ bảng (thủ công)** (có script `DROP TABLE` và hướng dẫn cập nhật khi thêm bảng mới).

**Quy ước:** `permissions.code` dùng dấu chấm, `code_check` dùng dấu hai chấm (JWT / `RequirePermission`). Khi thêm quyền mới, cập nhật `constants/permissions.go` rồi thêm dòng vào `000002` (hoặc migration mới) và chạy `go run ./cmd/syncpermissions` trên môi trường đã có dữ liệu.

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

1. Tạo cặp `000003_mô_tả.up.sql` và `000003_mô_tả.down.sql` (tăng số so với bản mới nhất).
2. **golang-migrate** tách lệnh theo `;` và **không** bỏ qua comment `--` — **không đặt `;` trong cùng dòng `-- ...`** (sẽ cắt nhầm). Tránh `;` trong chuỗi hoặc `$$ ... $$` nếu không chủ ý.
3. `go build` để embed file mới.

## Rollback (down)

App **không** tự chạy `.down.sql`. Rollback: chạy tay SQL down theo thứ tự ngược version, hoặc chỉnh `schema_migrations` khi hiểu rủi ro.

## RBAC phẳng

Không có hierarchy role. Quyền hiệu lực = hợp quyền từ mọi role của user + `user_permissions`. Sau khi user xác nhận email, app gán role `learner` (cần đã chạy `000002`).
