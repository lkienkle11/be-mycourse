# Database Schema

All tables are managed via **golang-migrate** with embedded SQL files in `migrations/`.  
Run `MIGRATE=1 go run .` to apply pending migrations (see `migrations/README.md`).

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

Migrations: `000007` (table creation) · `000010` (adds `refresh_token_session` column).

---

## `permissions`

RBAC permission definitions (flat — no hierarchy).

| Column | Type | Description |
|---|---|---|
| `id` | `BIGSERIAL` PK | |
| `name` | `VARCHAR(255)` UNIQUE NOT NULL | Human-readable name |
| `code_check` | `VARCHAR(255)` UNIQUE NOT NULL | Machine-readable code embedded in JWT permissions array |
| `description` | `TEXT` | |
| `created_at`, `updated_at` | `TIMESTAMPTZ` | |

---

## `roles`

Named role definitions.

| Column | Type | Description |
|---|---|---|
| `id` | `BIGSERIAL` PK | |
| `name` | `VARCHAR(255)` UNIQUE NOT NULL | |
| `description` | `TEXT` | |
| `created_at`, `updated_at` | `TIMESTAMPTZ` | |

---

## `role_permissions`

Many-to-many: roles ↔ permissions.

| Column | Type |
|---|---|
| `role_id` | `BIGINT` FK → `roles(id)` |
| `permission_id` | `BIGINT` FK → `permissions(id)` |

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
| `permission_id` | `BIGINT` FK → `permissions(id)` |

---

## Effective Permissions

A user's effective permissions = **union** of permissions from all assigned roles **plus** direct `user_permissions` grants.  
They are resolved at login time, embedded in the access token's `permissions` array, and checked by `middleware.RequirePermission`.

---

## Migration history

| Version | File | Description |
|---|---|---|
| 000001 | `rbac_schema` | Create `permissions`, `roles`, `role_permissions`, `user_roles`, `user_permissions` |
| 000002 | `rbac_seed` | Seed `rbac.manage`, `profile.read` permissions + `admin` role |
| 000003 | `rbac_flat` | Remove role hierarchy |
| 000004 | `rbac_remove_hierarchy_if_present` | Drop `role_closure` / `roles.parent_id` if present |
| 000005 | `permissions_code_check` | Add `code_check` column to `permissions` |
| 000006 | `admin_role_profile_read` | Grant `profile.read` to admin role |
| 000007 | `users_table` | Create `users` table |
| 000008 | `seed_all_permissions` | Seed full permission set |
| 000009 | `rbac_user_id_fk` | Add FK constraints from RBAC tables to `users(id)` |
| 000010 | `users_refresh_token_session` | Add `refresh_token_session JSONB` column to `users` |
