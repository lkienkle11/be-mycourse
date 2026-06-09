# PostgreSQL migrations

`*_up.sql` files are embedded into the binary (`migrations/embed.go`) and executed by version number prefix (`000001`, `000002`, ...). PostgreSQL stores the applied version in `schema_migrations`.

## Current migration chain

| File | Description |
|------|----------|
| `000001_schema` | RBAC + `users` (including `refresh_token_session`), seed permissions/roles. |
| `000002_taxonomy_domain` | Taxonomy: `course_levels`, `categories`, `tags`, etc. |
| `000003_media_metadata` | Creates **`media_files`** (upload gateway) + indexes. |
| `000004_media_orphan_safety` | Adds `media_files.row_version`, `content_fingerprint`; creates **`media_pending_cloud_cleanup`**. |
| `000005_media_bunny_response_fields` | Adds **`media_files.video_id`**, **`thumbnail_url`**, **`embeded_html`** (Bunny parity / API `UploadFileResponse`, see `docs/modules/media.md`). |
| `000006_taxonomy_user_media_refs` | Adds `categories.image_file_id` + `users.avatar_file_id` (FK → `media_files.id`); drops plain URL columns `image_url` / `avatar_url`; backfills FKs from matching URLs (`media_files.url` / `origin_url`). |
| `000007_registration_email_limits` | Adds **`users.registration_email_send_total`** (successful confirmation-email count while pending; reset on confirm). |
| `000008_media_metadata_json_storage` | Ensures **`media_files.metadata_json`** is JSONB server-side metadata storage; backfills typed keys such as **`duration_seconds`**, **`width_bytes`**, **`height_bytes`**, **`fps`**; adds GIN index **`idx_media_files_metadata_json_gin`**. |
| `000009_taxonomy_topics_outcomes_skills` | Renames `categories` → **`course_topics`** + `child_topics` JSONB, adds **`course_outcomes`** / **`course_skills`**, renames P18–P21 to **`topic:*`**, seeds P30–P37 **`course_outcome:*`** / **`course_skill:*`**. |
| `000010_role_modify_permissions` | Seeds P38–P40 **`sysadmin:modify`** / **`admin:modify`** / **`instructor:modify`** and grants by role tier (sysadmin -> all three, admin -> P39–P40, instructor -> P40). |
| `000011_audit_timestamps_bigint` | Converts audit columns **`created_at`**, **`updated_at`**, **`deleted_at`** (where present) from `TIMESTAMPTZ` to **`BIGINT`** Unix epoch seconds. **Must `DROP DEFAULT` before `ALTER TYPE`**, then `SET DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)`. |
| `000012_soft_delete_taxonomy_users_ban` | Adds **`deleted_at`** to 5 taxonomy tables + partial unique slug indexes; adds **`users.banned_until`** (Unix seconds ban-lift timestamp). |
| `000013_instructor_management` | Adds **`users.phone`**; creates instructor tables (applications, profiles, expertise, tickets, messages); seeds **P41–P58** + role grants. See **`docs/modules/instructor.md`**. |
| `000014_system_user_machine_binding` | Adds **`system_privileged_users.machine_secret`** for privileged CLI machine binding (existing rows default empty value). |
| `000015_instructor_expertise_soft_delete_compat` | Drift-safe compatibility patch for `instructor_expertise_topics` / `instructor_expertise_skills`: ensures `deleted_at`, normalizes `topic_id` / `skill_id` from legacy `course_topic_id` / `course_skill_id` when present, then rebuilds active-only unique indexes. Does **not** drop legacy columns — apply `000017` next on drifted DBs. |
| `000016_course_management` | Tables `courses`, `course_versions`, collaborator membership, version-scoped outline rows, sub-lesson payload tables, edit leases, learner enrollments, and stable-id progress records. See **`docs/modules/course.md`**. |
| `000017_instructor_expertise_drop_legacy_fk_cols` | Finalizes expertise junction schema on drifted DBs: backfills `topic_id` / `skill_id`, drops legacy `course_topic_id` / `course_skill_id`, sets canonical columns NOT NULL, and adds FK constraints on `topic_id` / `skill_id`. |
| `000018_instructor_tickets_soft_delete_compat` | Drift-safe compatibility patch for `instructor_tickets` / `instructor_ticket_messages`: ensures `deleted_at` and rebuilds partial status index (`WHERE deleted_at IS NULL`). |
| `000019_instructor_profiles_apps_soft_delete_compat` | Drift-safe patch for `instructor_profiles` / `instructor_applications`: ensures `deleted_at`, adds `id` PK on profiles when DB only has `user_id` PK, rebuilds partial unique indexes. |
| `000020_course_version_row_version_backfill` | Backfills `course_versions.row_version` from `0` to `1` for rows created before GORM explicitly set `RowVersion: 1` on insert (column default alone is overridden by zero-value inserts). |

**Drop all tables in SQL (correct FK order):** see `docs/database.md` -> **Drop All Tables**. When adding a new table, update that `DROP TABLE` list accordingly.

**Conventions:** `permission_id` uses `P{number}`; `permission_name` uses `resource:action` (JWT / `RequirePermission`). Full catalog **P1–P58** is in `internal/shared/constants/permissions.go`. When adding new permissions: update that file, optionally add migration seed, then run `go run ./cmd/syncpermissions` in existing environments. Role matrix lives in `internal/system/application/roles_permission.go` + `go run ./cmd/syncrolepermissions`. Full table/column details: **`docs/database.md`**.

**COMMENT / SQL strings with `golang-migrate`:** the runner splits files by **every** `;` (no SQL parser). Therefore, do **not** place `;` inside strings (`'...'`), inside **`$$...$$`**, etc. Use `;` only to terminate statements. In this repo, avoid `DO $$ ... $$` blocks entirely (see `000015` rewrite) and prefer plain DDL/DML statements. In `COMMENT ON ... IS '...'`, avoid `;` in comment text (use punctuation like commas or periods), e.g. `000007_registration_email_limits.up.sql`.

**Changing column type with DEFAULT:** when `ALTER COLUMN ... TYPE` cannot cast the old default (e.g. `TIMESTAMPTZ DEFAULT NOW()` -> `BIGINT`), run in this order: `DROP DEFAULT` -> `ALTER TYPE ... USING ...` -> `SET DEFAULT` (new value). Error `default for column "created_at" cannot be cast automatically to type bigint` means `DROP DEFAULT` was skipped; see `000011_audit_timestamps_bigint.up.sql`.

**Version >= 11 but column still `timestamptz`:** `schema_migrations` may have advanced while SQL `000011` did not fully apply. Verify via `information_schema` (see **`docs/deploy.md`** -> Troubleshooting), then re-run `psql ... -f migrations/000011_audit_timestamps_bigint.up.sql`.

## How to run migrations in current server flow

1. Configure PostgreSQL exactly like the running app (`config/app.yaml` + `.env` -> `[database]` section).
2. In **PowerShell**:

```powershell
$env:MIGRATE = "1"
go run .
```

Or run the built binary:

```powershell
$env:MIGRATE = "1"
.\mycourse-io-be.exe
```

3. With `MIGRATE=1`, the app runs **up migration and then continues normal startup** (Gin still starts).

4. Roll back by a specific migration file (PowerShell):

```powershell
$env:MIGRATE = "2"
$env:MIGRATE_VERSION_FILE = "000016_course_management.down.sql"
go run .
```

`MIGRATE=2` is rollback-only: the app parses version from `MIGRATE_VERSION_FILE`, migrates DB to `version-1`, then exits.

## Add a new migration

1. Create a pair: `00000N_description.up.sql` and `00000N_description.down.sql` (increment from latest version).
2. **golang-migrate** splits on `;` and does **not** safely ignore inline comment content — do **not** put `;` in `-- ...` comment lines. Avoid extra `;` anywhere (including strings / dollar-quoted blocks); see **COMMENT** note above.
3. Run `go build` to embed the new SQL files.
4. Update this table in **`migrations/README.md`** and **`docs/database.md`** (migration history + detailed table notes when needed).

## Rollback (down)

App supports rollback via env:

- `MIGRATE=2`
- `MIGRATE_VERSION_FILE=<file>.down.sql`

Rules:

- Accepts only `.down.sql` filenames.
- Reads version from the numeric filename prefix (e.g. `000016_*`).
- Down target = `version - 1` (e.g. `000016...down.sql` targets version 15).
- Refuses rollback when `schema_migrations` is `dirty`.
- Refuses rollback when current version is not greater than target.

## Flat RBAC model

No role hierarchy. Effective permissions = union of all role permissions for the user + `user_permissions`. After email confirmation, the app assigns role `learner` (requires `000001` to be applied).
