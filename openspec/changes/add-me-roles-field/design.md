## Context

`GetMe` (`internal/auth/application/service.go`) reads from and writes to a Redis cache (`getCachedMe`/`setCachedMe`/`InvalidateUserMeCache` in `internal/auth/application/service_cache.go`) and depends on an injected `permReader.PermissionCodesForUser` port for permission codes. RBAC role lookups (`RBACService.ListRolesForUser`) and role/direct-permission mutations are exposed through the admin-only RBAC HTTP surface (`internal/rbac/delivery/handler.go`). The server composition root injects the existing `AuthService.InvalidateUserMeCache` implementation into that handler, so successful binding changes evict the cached projection without duplicating Redis-key logic. See proposal.md for why this field is added.

## Goals / Non-Goals

**Goals:**
- Add a `roles []string` field to `/me`, sourced from RBAC, ordered by fixed priority, display-only.
- Keep the cached `/me` payload correct after role/direct-permission assignment or removal, following the same invalidation pattern already used for instructor-triggered cache busts.
- Keep backward-compatible deserialization for cache entries written before this field existed.

**Non-Goals:**
- No change to permission-based authorization, middleware, or any `RequirePermission`/`HasPermission` check.
- No new admin UI or endpoint for managing roles (assignment already exists via the RBAC admin API).
- No dynamic/config-driven priority ranking — the ranking is a hardcoded ordered list in code, extended manually when new role names are introduced.
- No i18n/label mapping on the backend — FE maps raw role names to display labels.

## Decisions

**1. Wire `auth` → `rbac` through an auth-owned role-name reader port.**
`AuthService` already takes a `permReader` interface for permission codes. Add a sibling `RoleNameReader`-style interface in `internal/auth/application` whose method returns only `[]string`, then implement it in the `internal/server` composition root by calling the existing `RBACService.ListRolesForUser` once and mapping each returned role to its `Name`. This keeps RBAC domain types out of the auth bounded context, avoids an import cycle, prevents permission payloads from leaking into the `/me` projection, and matches the existing dependency-injection style.
*Alternative considered*: have `auth` query the `user_roles` table directly. Rejected — bypasses the RBAC bounded context's repository abstraction that `ListRolesForUser` already provides, duplicating query logic.

**2. Fixed priority ranking as an ordered slice constant in the `auth` application layer.**
Define an ordered list, e.g. `var roleRankOrder = []string{"sysadmin", "admin", "instructor", "learner"}`, and sort each user's role names by index in that list (unranked names get an index of `len(roleRankOrder)` so they sort after all ranked names, in original/stable order among themselves). This lives next to `buildMeProfile` since it's a `/me`-presentation concern, not an RBAC domain concept — RBAC itself has no notion of role ranking.
*Alternative considered*: store a `priority` column on the `roles` table. Rejected per explicit decision in explore mode — no dynamic ranking wanted; a code change is the intended process for adding new role names.

**3. `MeProfile.Roles []string` and `MeResponse.Roles []string` are plain string slices, not RBAC domain roles.**
Only the role `Name` is needed for display; carrying a full RBAC role (with nested `Permissions`) through the auth service, cache, and HTTP response would leak RBAC internals into the `/me` display projection and bloat the cached payload. The server adapter maps RBAC roles to names at the bounded-context boundary; `buildMeProfile` receives only the resulting names.

**4. Cache invalidation: inject the existing auth cache invalidator into the RBAC HTTP handler.**
The smallest delivery-owned `MeCacheInvalidator` interface is implemented by the existing `AuthService` and injected from `internal/server`. `assignUserRole`, `removeUserRole`, `assignUserPermission`, and `removeUserPermission` call it only after a successful persistence operation. No second cache adapter or Redis-key implementation is introduced. Non-HTTP role mutation callers remain covered by the existing instructor invalidator, confirmation cache clear, learner self-heal, or pre-warm OAuth ordering.

**5. Backward-compatible cache deserialization: a missing `roles` key is a cache miss.**
Adding the field remains JSON-compatible: an older payload unmarshals with `Roles == nil`. `getCachedMe` must treat that nil slice as a stale-shape cache miss so `GetMe` reloads the current role bindings and rewrites the cache. A freshly built profile with no roles must store a non-nil empty slice, which remains distinguishable from the legacy missing field after JSON decoding. `toMeResponse` also normalizes nil defensively so no code path can emit `"roles": null`.

**6. Post-review cache correctness covers direct permission mutations too.**
`MeProfile` contains both roles and effective permissions, so the existing `MeCacheInvalidator` is called after successful direct-permission assignment/removal as well as role assignment/removal. The same post-persistence ordering applies: validation, lookup, or repository failures must not evict a valid cache entry.

**7. Keep dormant authorization infrastructure minimal until a provider exists.**
The authorization base currently has no registered `PolicyProvider` and no runtime consumer. Startup keeps only the action-catalog synchronization boundary and returns an error, rather than constructing and returning an unused `Authorizer`/`GrantService`/`EffectiveActionProjector` bundle. The unconsumed context-carried GORM transaction path is removed; `ReplaceMany` remains atomic through its repository-owned GORM transaction. A future provider may introduce a cross-repository transaction contract together with its production caller and integration coverage.

**8. JWT permission revocation latency remains an explicit accepted boundary.**
Global permission claims are evaluated from the access token. Removing a role or direct permission updates `/me` immediately through cache invalidation, but an already-issued token may still authorize a permission it contains until refresh or expiry. Resource-scoped grants, once a provider exists, remain live database reads. This distinction is documented rather than hidden by the `/me` cache fix.

## Risks / Trade-offs

- **[Risk]** Adding an RBAC read to every `GetMe` cache-miss path adds one more query/service call → **[Mitigation]** Reuses the existing Redis cache for `/me`; the extra RBAC read only happens on cache miss, same cost class as the existing permission-codes read it sits beside.
- **[Risk]** Forgetting cache invalidation after any role or direct-permission mutation leaves `/me` stale for the cache TTL → **[Mitigation]** Every RBAC binding handler reuses `InvalidateUserMeCache` after persistence, with tests proving failures do not evict.
- **[Risk]** Hardcoded ranking silently drops a future role to the bottom without anyone noticing → **[Mitigation]** Accepted explicitly by the user during design (explore mode decision): future new role names require a manual code change to `roleRankOrder`; unranked names still appear in `roles`, just ordered last, so nothing is hidden.
- **[Risk]** FE or another BE consumer starts using `roles` for access control despite the display-only contract → **[Mitigation]** Spec requirement explicitly forbids this ("`roles` is excluded from authorization logic"); code comment on the `Roles` field in `MeProfile`/`MeResponse` should state the same constraint inline.
