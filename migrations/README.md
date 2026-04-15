# PostgreSQL migrations

Các file `*_up.sql` được **nhúng (embed)** vào binary (`migrations/embed.go`) và chạy theo **số version** ở đầu tên file (`000001`, `000002`, …). Bảng `schema_migrations` trên Postgres ghi version đã áp dụng.

## Chuỗi hiện tại (đã gộp)

| File | Nội dung |
|------|----------|
| `000001_schema` | Tạo toàn bộ schema RBAC + users, dùng `permissions.action`, đồng thời seed sẵn 12 permissions + 4 roles (`sysadmin`, `admin`, `instructor`, `learner`) và ma trận `role_permissions`. |

Chuỗi cũ nhiều bước (`000002` trở lên) đã bỏ để tối giản bootstrap. Với DB cũ đã chạy các version trước, nên reset DB + `schema_migrations`.

**Xóa toàn bộ bảng bằng SQL (đúng thứ tự FK):** xem `docs/database.md` — mục **Xóa toàn bộ bảng (thủ công)** (có script `DROP TABLE` và hướng dẫn cập nhật khi thêm bảng mới).

**Quy ước:** `permissions.code` dùng dấu chấm, `permissions.action` dùng dấu hai chấm (JWT / `RequirePermission`). Khi thêm quyền mới, cập nhật `constants/permissions.go`, tạo migration mới và chạy `go run ./cmd/syncpermissions` trên môi trường đã có dữ liệu.

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

1. Tạo cặp `000002_mô_tả.up.sql` và `000002_mô_tả.down.sql` (tăng số so với bản mới nhất).
2. **golang-migrate** tách lệnh theo `;` và **không** bỏ qua comment `--` — **không đặt `;` trong cùng dòng `-- ...`** (sẽ cắt nhầm). Tránh `;` trong chuỗi hoặc `$$ ... $$` nếu không chủ ý.
3. `go build` để embed file mới.

## Rollback (down)

App **không** tự chạy `.down.sql`. Rollback: chạy tay SQL down theo thứ tự ngược version, hoặc chỉnh `schema_migrations` khi hiểu rủi ro.

## RBAC phẳng

Không có hierarchy role. Quyền hiệu lực = hợp quyền từ mọi role của user + `user_permissions`. Sau khi user xác nhận email, app gán role `learner` (cần đã chạy `000001`).
