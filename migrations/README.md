# PostgreSQL migrations (RBAC)

Các file `*_up.sql` được **nhúng (embed)** vào binary và chạy theo **số version** ở đầu tên file (ví dụ `000001`, `000002`). Bảng `schema_migrations` trên Postgres ghi lại version đã chạy để không chạy lại.

## Cách chạy migration với Gin / server hiện tại

1. Cấu hình kết nối Postgres giống lúc chạy app (`config/app.yaml` + `.env` — mục `[database]`).
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

3. Ứng dụng sẽ: khởi tạo DB → áp dụng các file `.up.sql` còn thiếu → **sau đó vẫn khởi động Gin** như bình thường. Muốn **chỉ migrate rồi thoát**, có thể tạm dùng `Ctrl+C` ngay sau log thành công, hoặc thêm flag riêng sau này.

Biến môi trường `MIGRATE=1` là cách dự án đang dùng trong `main.go` (không phải tính năng có sẵn của Gin — Gin chỉ là HTTP router; migration gọi `database/sql` trước khi `router.Run`).

## Thêm migration mới

1. Tạo cặp file (khuyến nghị): `000003_mô_tả.up.sql` và `000003_mô_tả.down.sql`.
2. Tăng số `000003` so với bản mới nhất hiện có.
3. Trong `.up.sql`: tránh dùng `;` bên trong chuỗi SQL hoặc thân hàm dạng `$$ ... $$` (runner tách lệnh theo `;` ngoài dấu nháy đơn).
4. Build lại (`go build`) để embed file mới.

## Rollback (down)

Chạy thủ công nội dung file `*_down.sql` trên Postgres (theo thứ tự ngược version), hoặc xóa dòng tương ứng trong `schema_migrations` **chỉ khi bạn hiểu rủi ro** — hiện app **không** tự chạy `.down.sql`.

## Dữ liệu seed

`000002_rbac_seed.up.sql` tạo permission `rbac.manage`, `profile.read`, role `admin` và gán quyền (idempotent với `ON CONFLICT`).

## RBAC phẳng (role–permission–user)

- `000001` tạo `permissions`, `roles`, `role_permissions`, `user_roles`, `user_permissions` (quyền gán thẳng cho user, bổ sung cho quyền từ role).
- `000003_rbac_flat.up.sql` và (nếu DB cũ đã từng có hierarchy) `000004_rbac_remove_hierarchy_if_present.up.sql` gỡ `role_closure` / `roles.parent_id` nếu còn sót. Không còn kế thừa role cha–con; quyền hiệu lực = hợp của quyền từ các role + `user_permissions`.
