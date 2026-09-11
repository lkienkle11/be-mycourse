## Why

The FE `/me` response exposes `Permissions []string` but no role information at all, so the FE cannot show a user's job title (chức danh) anywhere in the UI. Users see no indication of whether they are an admin, instructor, or learner. RBAC already models roles (`roles`/`user_roles` tables, many-to-many), so the data exists; it is just never surfaced through `/me`.

## What Changes

- Add a `roles []string` field to the `GET /api/v1/me` response, populated from the RBAC `user_roles` binding for the authenticated user.
- Role values are raw, English, lowercase names exactly as stored in RBAC (`sysadmin`, `admin`, `instructor`, `learner`, ...) — no localized labels are generated server-side; the FE is responsible for mapping names to display labels/i18n.
- The array is sorted in descending order of a fixed, hardcoded priority ranking (`sysadmin` > `admin` > `instructor` > `learner`). Extending the ranking for a future role name requires a code change, not data-driven configuration.
- A user with no role bindings gets `"roles": []` (never `null`, never omitted).
- **Display-only contract**: `roles` exists purely for FE presentation. It MUST NOT be used by FE or BE for authorization decisions — permission codes (`/me/permissions`, `PermissionCodesForUser`) remain the sole source of truth for access control and are unchanged by this proposal.
- The Redis-cached `/me` projection (`MeProfile`) gains this field; existing cache entries written before this change (which lack the field) must not break deserialization, and role/direct-permission assignment or removal must invalidate the cached `/me` entry so roles and effective permissions cannot go stale.

## Capabilities

### New Capabilities
- `auth/me-roles-display`: Display-only role names surfaced on the `/me` profile endpoint, sourced from RBAC role bindings, ordered by a fixed priority ranking, explicitly excluded from authorization decisions.

### Modified Capabilities
(none — no existing capability specs are archived under `openspec/specs/` yet for `/me` or RBAC, so this is additive only)

## Impact

- **Affected code**: `internal/auth/domain/user.go` (`MeProfile`), `internal/auth/delivery/dto.go` (`MeResponse`), `internal/auth/delivery/handler.go` (`toMeResponse`), `internal/auth/application/service.go` (`AuthService.GetMe`, `buildMeProfile`), `internal/auth/application/service_cache.go` (Redis cache read/write for `MeProfile`), `internal/rbac/application/service.go` (`ListRolesForUser`, consumed via a new interface dependency in `auth`).
- **Affected APIs**: `GET /api/v1/me` response shape gains one additive field (`roles`); no breaking change to existing consumers since it's a new optional field.
- **Dependencies**: `auth` bounded context takes a new read dependency on the `rbac` bounded context's role-listing capability (already exists, just not wired into `auth`).
- **Cache**: Redis-cached `MeProfile` payload gains a field; requires backward-compatible deserialization and a cache-invalidation hook on RBAC role/direct-permission assign/remove operations for the affected user.
- **No changes** to authorization/permission-check logic, middleware, or any `RequirePermission`-style guards.
