# Database Schema

All tables are managed via **golang-migrate** with embedded SQL files in `migrations/`.  
Run `MIGRATE=1 go run .` to apply pending migrations (see `migrations/README.md`).

## Table of Contents

- [Table `users`](#users)
- [Table `permissions`](#permissions)
- [Table `roles`](#roles)
- [Table `role_permissions`](#role_permissions)
- [Table `user_roles`](#user_roles)
- [Table `user_permissions`](#user_permissions)
- [System Tables](#system-tables)
- [Effective Permissions](#effective-permissions)
- [Migration History](#migration-history)
- [Drop All Tables (manual reset)](#drop-all-tables-manual-reset)

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

The `refresh_token_session` column schema is defined in migration `000001_schema` (see `migrations/README.md`).

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

**Default permission sets** (seeded in `000001_schema.up.sql`, rebuilt from `constants/roles_permission.go` by `cmd/syncrolepermissions`):
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

For details and notes on resetting the DB when changing the migration sequence, see `migrations/README.md`.

---

## System Tables

Isolated tables with no foreign keys to the application RBAC / users tables.

### `system_app_config`

Singleton configuration row (always `id = 1`). Secrets are managed out-of-band (SQL or tooling).

| Column | Type | Description |
|---|---|---|
| `id` | `INTEGER` PK CHECK (id = 1) | Always 1 — singleton constraint |
| `app_cli_system_password` | `TEXT` NOT NULL DEFAULT `''` | CLI registration password (plaintext) |
| `app_system_env` | `TEXT` NOT NULL DEFAULT `''` | HMAC key for system credential derivation |
| `app_token_env` | `TEXT` NOT NULL DEFAULT `''` | JWT secret for system access tokens |
| `updated_at` | `TIMESTAMPTZ` | Last update timestamp |

### `system_privileged_users`

Privileged system operators. Credentials are HMAC-derived using `app_system_env` at write and login time — raw values are never stored.

| Column | Type | Description |
|---|---|---|
| `id` | `BIGSERIAL` PK | |
| `username_secret` | `TEXT` NOT NULL UNIQUE | HMAC-hex of username with `app_system_env` |
| `password_secret` | `TEXT` NOT NULL | HMAC-hex of password with `app_system_env` |
| `created_at` | `TIMESTAMPTZ` | |

---

## Drop All Tables (manual reset)

To **drop all tables** created by the current migration (e.g. to reset a dev environment), run the following commands **in order** on Postgres. Order follows foreign key dependency: child tables (junction / migrate metadata) first, parent tables last.

```sql
DROP TABLE public.schema_migrations;
DROP TABLE public.system_privileged_users;
DROP TABLE public.system_app_config;
DROP TABLE public.role_permissions;
DROP TABLE public.user_permissions;
DROP TABLE public.user_roles;
DROP TABLE public.roles;
DROP TABLE public.permissions;
DROP TABLE public.users;
```

**Maintenance note:** When adding a **new table** in a migration, update this list: insert `DROP TABLE public.<table_name>;` at the appropriate position (usually **before** any table it references via FK). Keep `schema_migrations` at the **top** of the list as it has no FK dependents.

**Automated tests:** DB-backed integration suites belong under repository root **`tests/`** (see `tests/README.md` and root `README.md` **Testing**).
