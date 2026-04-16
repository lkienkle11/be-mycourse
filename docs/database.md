# Database Schema

All tables are managed via **golang-migrate** with embedded SQL files in `migrations/`.  
Run `MIGRATE=1 go run .` to apply pending migrations (see `migrations/README.md`).

## Mục nội dung

- [Bảng `users`](#users)
- [Bảng `permissions`](#permissions)
- [Bảng `roles`](#roles)
- [Bảng `role_permissions`](#role_permissions)
- [Bảng `user_roles`](#user_roles)
- [Bảng `user_permissions`](#user_permissions)
- [Effective Permissions](#effective-permissions)
- [Migration history](#migration-history)
- [Xóa toàn bộ bảng (thủ công)](#xoa-toan-bo-bang-thu-cong)

---

## `users`

Stores application accounts.  Passwords are bcrypt-hashed.  
`user_code` (UUIDv7) is the external-facing identifier used in RBAC tables; `id` (BIGSERIAL) is used for internal joins.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `BIGSERIAL` | PK | Internal numeric primary key |
| `user_code` | `UUID` | UNIQUE NOT NULL | External identifier (UUIDv7), used in RBAC |
| `email` | `VARCHAR(255)` | UNIQUE NOT NULL | Login email |
| `hash_password` | `VARCHAR(255)` | NOT NULL | bcrypt hash |
| `display_name` | `VARCHAR(255)` | NOT NULL DEFAULT `''` | Display name |
| `avatar_url` | `TEXT` | NOT NULL DEFAULT `''` | Profile picture URL |
| `is_disable` | `BOOLEAN` | NOT NULL DEFAULT `FALSE` | Account disabled flag |
| `email_confirmed` | `BOOLEAN` | NOT NULL DEFAULT `FALSE` | Email verification status |
| `confirmation_token` | `VARCHAR(128)` | nullable | One-time email confirmation token |
| `confirmation_sent_at` | `TIMESTAMPTZ` | nullable | When the confirmation email was sent |
| `refresh_token_session` | `JSONB` | NOT NULL DEFAULT `'{}'` | Active device sessions map (see below) |
| `created_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `NOW()` | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL DEFAULT `NOW()` | |
| `deleted_at` | `TIMESTAMPTZ` | nullable, indexed | Soft-delete timestamp (GORM) |

### `refresh_token_session` structure

A JSONB map where each key is a **128-char hex session string** and each value is a session entry:

```json
{
  "<session_string_128_hex>": {
    "refresh_token_uuid":    "uuid-v4",
    "remember_me":           false,
    "refresh_token_expired": "2026-05-03T12:00:00Z"
  }
}
```

- Maximum **5 entries** per user. On overflow the entry with the earliest `refresh_token_expired` is evicted.
- Writes that change the entry count run inside a **transaction** (`models.AddRefreshSession`).
- In-place rotation (same key, new UUID + expiry) uses a lockless `jsonb_set` update (`models.SaveRefreshSession`).

Schema và cột `refresh_token_session` nằm trong migration `000001_schema` (xem `migrations/README.md`).

---

## `permissions`

RBAC permission definitions (flat — no hierarchy). The primary key is the **catalog id** (`permission_id`, e.g. `P1`), not a surrogate bigint.

| Column | Type | Description |
|---|---|---|
| `permission_id` | `VARCHAR(10)` PK | Stable id (`P{number}`), aligned with struct tag `perm_id` on `constants.AllPermissions` |
| `permission_name` | `VARCHAR(50)` UNIQUE NOT NULL | Runtime string in the JWT `permissions` claim and `middleware.RequirePermission` (`resource:action`, e.g. `course:read`) |
| `description` | `VARCHAR(512)` NOT NULL DEFAULT `''` | |
| `created_at`, `updated_at` | `TIMESTAMPTZ` | |

**Sync:** `cmd/syncpermissions` upserts `permission_name` (and inserts missing rows) for every entry from `constants.AllPermissionEntries()` **by `permission_id`**. Extra rows present only in the database are **left unchanged**.

---

## `roles`

Named role definitions. Application constants live in `constants/roles.go` (`sysadmin`, `admin`, `instructor`, `learner`).

| Column | Type | Description |
|---|---|---|
| `id` | `BIGSERIAL` PK | |
| `name` | `VARCHAR(64)` UNIQUE NOT NULL | |
| `description` | `TEXT` | |
| `created_at`, `updated_at` | `TIMESTAMPTZ` | |

**Default permission sets** (seed trong `000001_schema.up.sql`, rebuilt from `constants/roles_permission.go` by `cmd/syncrolepermissions`):
- `sysadmin`: `P1`–`P13` (full catalog)
- `admin`: profile + course + user CRUD, **không** `P9` (`course_instructor:read`)
- `instructor`: `P1`, `P5`–`P7`, `P9`, `P10`
- `learner`: `P1`, `P5`, `P10`

---

## `role_permissions`

Many-to-many: roles ↔ permissions. `permission_id` here is the **string PK** on `permissions`, not a bigint.

| Column | Type |
|---|---|
| `role_id` | `BIGINT` FK → `roles(id)` ON DELETE CASCADE |
| `permission_id` | `VARCHAR(10)` FK → `permissions(permission_id)` ON DELETE CASCADE, ON UPDATE CASCADE |

---

## `user_roles`

Many-to-many: users ↔ roles.

| Column | Type |
|---|---|
| `user_id` | `BIGINT` FK → `users(id)` |
| `role_id` | `BIGINT` FK → `roles(id)` |

---

## `user_permissions`

Direct permission grants (supplement role-based permissions).

| Column | Type |
|---|---|
| `user_id` | `BIGINT` FK → `users(id)` |
| `permission_id` | `VARCHAR(10)` FK → `permissions(permission_id)` ON DELETE CASCADE, ON UPDATE CASCADE |

---

## Effective Permissions

A user's effective permissions = **union** of permissions from all assigned roles **plus** direct `user_permissions` grants.  
They are resolved at login time, embedded in the access token's `permissions` array as **`permission_name`** strings (colon form, e.g. `course:read`), and checked by `middleware.RequirePermission` against the same values from `constants.AllPermissions`.

---

## Migration history

| Version | File | Description |
|---|---|---|
| 000001 | `schema` | Create `permissions` (`permission_id` PK + `permission_name`), `roles`, `role_permissions`, `users` (with `refresh_token_session`), `user_roles`, `user_permissions`, and seed 13 permissions + 4 default roles + `role_permissions` matrix |

Chi tiết và lưu ý reset DB khi đổi chuỗi migration: `migrations/README.md`.

---

<a id="xoa-toan-bo-bang-thu-cong"></a>

## Xóa toàn bộ bảng (thủ công)

Để **xóa hết** các bảng do migration hiện tại tạo ra (ví dụ reset môi trường dev), chạy các lệnh **đúng thứ tự** sau trên Postgres. Thứ tự phụ thuộc khóa ngoại: bảng con (junction / migrate metadata) trước, bảng cha sau — nếu đảo thứ tự sẽ lỗi FK.

```sql
DROP TABLE public.schema_migrations;
DROP TABLE public.role_permissions;
DROP TABLE public.user_permissions;
DROP TABLE public.user_roles;
DROP TABLE public.roles;
DROP TABLE public.permissions;
DROP TABLE public.users;
```

**Lưu ý bảo trì:** Khi thêm **bảng mới** trong migration, cập nhật lại mục này: chèn `DROP TABLE public.<tên_bảng>;` ở vị trí thích hợp (thường là **trước** mọi bảng mà nó tham chiếu tới). Giữ `schema_migrations` ở **đầu** danh sách như trên là hợp lý vì bảng đó không bị bảng khác FK tới.
