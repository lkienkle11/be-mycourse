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
| `000007_registration_email_limits` | Cột **`users.registration_email_send_total`** (đếm email xác nhận gửi thành công khi pending; reset khi confirm). |
| `000008_media_metadata_json_storage` | Đảm bảo **`media_files.metadata_json`** là JSONB server-side metadata store, backfill typed keys như **`duration_seconds`**, **`width_bytes`**, **`height_bytes`**, **`fps`**, và thêm GIN index **`idx_media_files_metadata_json_gin`** để query provider metadata khi cần. |
| `000009_taxonomy_topics_outcomes_skills` | Đổi `categories` → **`course_topics`** + `child_topics` JSONB, bảng **`course_outcomes`** / **`course_skills`**, đổi P18–P21 thành **`topic:*`**, seed P30–P37 **`course_outcome:*`** / **`course_skill:*`**. |
| `000010_role_modify_permissions` | Seed P38–P40 **`sysadmin:modify`** / **`admin:modify`** / **`instructor:modify`** và gán theo role tier (sysadmin → cả ba, admin → P39–P40, instructor → P40). |
| `000011_audit_timestamps_bigint` | Đổi cột audit **`created_at`**, **`updated_at`**, **`deleted_at`** (nơi có) từ `TIMESTAMPTZ` sang **`BIGINT`** Unix epoch seconds. **Bắt buộc `DROP DEFAULT` trước `ALTER TYPE`** (Postgres không cast `DEFAULT NOW()` sang `BIGINT` tự động), rồi `SET DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)`. |
| `000012_soft_delete_taxonomy_users_ban` | **`deleted_at`** trên 5 bảng taxonomy + partial unique slug indexes, cột **`users.banned_until`** (Unix seconds, thời điểm hết ban). |
| `000013_instructor_management` | **`users.phone`**; bảng instructor (applications, profiles, expertise, tickets, messages); seed **P41–P58** + gán role. Xem **`docs/modules/instructor.md`**. |

**Xóa toàn bộ bảng bằng SQL (đúng thứ tự FK):** xem `docs/database.md` — mục **Drop All Tables**; khi thêm bảng mới cập nhật danh sách `DROP TABLE` tương ứng.

**Quy ước:** `permission_id` dạng `P{number}`; `permission_name` dạng `resource:action` (JWT / `RequirePermission`). Catalog đầy đủ **P1–P58** nằm trong `internal/shared/constants/permissions.go`. Khi thêm quyền mới: cập nhật file đó, (tuỳ chọn) migration seed, rồi `go run ./cmd/syncpermissions` trên môi trường đã có dữ liệu. Ma trận role: `internal/system/application/roles_permission.go` + `go run ./cmd/syncrolepermissions`. Chi tiết bảng/cột: **`docs/database.md`**.

**COMMENT / chuỗi SQL và `golang-migrate`:** runner tách file theo **mọi** dấu `;` (không hiểu cú pháp SQL). Vì vậy **không** được có `;` bên trong chuỗi (`'…'`), trong **`$$…$$`**, v.v. — chỉ dùng `;` làm kết thúc từng câu lệnh. Trong `COMMENT ON … IS '…'`, viết mô tả bằng dấu chấm/phẩy thay cho `;` — ví dụ `000007_registration_email_limits.up.sql`.

**Đổi kiểu cột có DEFAULT:** khi `ALTER COLUMN … TYPE` mà default cũ không cast được (ví dụ `TIMESTAMPTZ DEFAULT NOW()` → `BIGINT`), chạy theo thứ tự: `DROP DEFAULT` → `ALTER TYPE … USING …` → `SET DEFAULT` mới. Lỗi `default for column "created_at" cannot be cast automatically to type bigint` nghĩa là thiếu bước `DROP DEFAULT` — xem `000011_audit_timestamps_bigint.up.sql`.

**Version ≥ 11 nhưng cột vẫn `timestamptz`:** `schema_migrations` có thể đã tăng version mà SQL `000011` chưa chạy hết. Verify bằng `information_schema` (xem **`docs/deploy.md`** — Troubleshooting), rồi chạy lại `psql … -f migrations/000011_audit_timestamps_bigint.up.sql`.

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
2. **golang-migrate** tách lệnh theo `;` và **không** bỏ qua comment `--` — **không đặt `;` trong cùng dòng `-- ...`** (sẽ cắt nhầm). Tránh mọi `;` “thừa” trong file (kể cả trong chuỗi / dollar-quote); xem mục **COMMENT** ở trên.
3. `go build` để embed file mới.
4. Cập nhật bảng trên trong **`migrations/README.md`** và **`docs/database.md`** (Migration history + bảng chi tiết nếu cần).

## Rollback (down)

App **không** tự chạy `.down.sql`. Rollback: chạy tay SQL down theo thứ tự ngược version, hoặc chỉnh `schema_migrations` khi hiểu rủi ro.

## RBAC phẳng

Không có hierarchy role. Quyền hiệu lực = hợp quyền từ mọi role của user + `user_permissions`. Sau khi user xác nhận email, app gán role `learner` (cần đã chạy `000001`).
