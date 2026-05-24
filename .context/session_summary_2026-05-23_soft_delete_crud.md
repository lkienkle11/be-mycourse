# Session summary — soft-delete CRUD convention (2026-05-23)

## Goal

Implement soft-delete CRUD convention for **taxonomy** (5 resources) and **auth users** (`/me`), with migration, shared helpers, docs sync, and verification.

## Completed

### Shared foundation
- `internal/shared/gormx/soft_delete_scope.go` — `ScopeActiveOnly`, `ScopeIncludeDeleted`
- `domain.TaxonomyFilter.IncludeDeleted` for `GET .../full` lists

### Migration `000012`
- `deleted_at BIGINT` on `course_levels`, `course_topics`, `course_outcomes`, `course_skills`, `tags`
- Partial unique slug indexes (`uix_*_slug_active`) + partial `deleted_at` indexes
- `users.banned_until BIGINT NULL`
- `migrations/README.md` updated

### Taxonomy module
- Domain: `DeletedAt *int64` on all five entities
- Infra: `SoftDelete` / `HardDelete`; active-only list/get via `ScopeActiveOnly`
- Application: default delete = soft; hard delete for topics/outcomes runs orphan image cleanup only on hard path
- Delivery: `GET /taxonomy/{resource}/full`, `DELETE /:id/hard` (static routes before param routes)

### Auth module
- `BannedUntil *int64` on user; `checkUserAccessible` in application layer, `ErrUserBanned` (`4012`)
- `FindByID` active-only; login/refresh/GetMe/PatchMe guards
- `DELETE /api/v1/me/hard` permanent removal

### Docs synced (English)
- `docs/patterns.md` — CRUD soft-delete section; `checkUserAccessible` in **`application/service_access.go`**
- `docs/architecture.md`, `docs/api-overview.md`, `docs/router.md`, `docs/database.md`, `docs/curl_api.md`
- `docs/modules/taxonomy.md`, `docs/modules/auth.md`, `docs/modules.md`
- `docs/requirements.md` (FR-1.3 login ban, FR-2.1c delete /me/hard)
- `docs/data-flow.md` (login, refresh, /me, delete flows)

## Deploy note

Run migration before deploying:

```bash
MIGRATE=1 go run .
```

## Out of scope (unchanged)

- RBAC hard-delete junction behavior
- Media route convention
- Admin user CRUD / ban-set API
- `GET /me/full`

## GitNexus impact notes (pre-edit)

- `taxonomyList`: LOW — direct callers in repo `List` methods only
- `FindByID` (auth): CRITICAL — GetMe, UpdateMe, SoftDeleteUser, RefreshSession, loadUserForLogin; all updated with access guards / active-only lookup
