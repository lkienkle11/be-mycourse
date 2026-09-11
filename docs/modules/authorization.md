# Authorization

`internal/authorization` is a shared, domain-independent IAM-lite bounded context. It does not import Course and exposes concrete shared components:

- `Registry`: validates each `PolicyProvider`, rejects duplicate action names/providers, and maps a scoped action to its resource type and global RBAC boundary.
- `GormGrantRepository`: persists the global action catalog and generic user/resource grants in `authorization_actions` and `authorization_grants`, and resolves resource-scoped role bindings (`authorization_role_bindings`/`authorization_role_actions`) into the same grant shape.
- `Authorizer`: evaluates `Principal + Action + Resource + Context` and returns a structured Allow/Deny decision.
- `EffectiveActionProjector`: resolves trusted principals in one RBAC batch, loads grants once, and projects effective actions through the same private decision path as `Authorizer`.
- `GrantService`: the only application path for grant, replace, revoke, batch replace, and resource cleanup.

## Grant statement semantics

`Grant()` appends one independent policy statement. Multiple statements may use the same principal, action, resource, and effect, including identical statements or statements with different conditions and validity windows. The database therefore does not enforce tuple uniqueness; each statement has its own UUID and audit history.

Expiration and explicit revocation are separate states. An expired statement remains with `revoked_at = NULL`, no longer matches authorization decisions, and does not prevent a new statement with the same tuple from being created. `Revoke` revokes every matching statement for the requested action/effect/resource; v1 does not expose revoke-by-grant-ID.

A bulk-replace caller uses `ReplaceMany`, not additive `Grant()`, to revoke a managed ALLOW set and insert the requested replacement set in one transaction, so repeating the same bulk request does not accumulate active grants for the same principal/resource/managed-action set.

`GrantService.requireManager` (shared by `Grant`, `Replace`, `ReplaceMany`, `Revoke`, and `RevokePrincipalResource`) denies grant management unless the issuer holds the global boundary permission for every action the registry marks grantable for the target resource type, in addition to the resource level authority each `PolicyProvider.CanManageGrants` implementation checks itself. This is enforced inside the authorization module itself, independent of whatever HTTP route level permission middleware happens to guard the caller, so it does not depend on every future caller remembering to add that middleware. `GrantService.RevokeResource`, the resource lifecycle cleanup path used when a resource is deleted, has no issuing principal and is exempt from this check.

`GormGrantRepository.ReplaceMany` owns its GORM transaction, so its revoke-and-insert replacement remains atomic. The base does not expose a context-carried transaction helper while there is no production caller that needs to coordinate authorization writes with another repository; such a contract must be introduced together with its first real caller and integration coverage.

## Decision order

1. Invalid principal, unknown action, or resource mismatch denies.
2. Missing global `boundary_permission` denies.
3. The domain provider validates server-created facts/context and may hard-deny or provide a built-in allow.
4. Active, unexpired grants whose conditions match are loaded.
5. Any matching explicit `DENY` wins.
6. Provider built-in allow or a matching `ALLOW` grant allows; otherwise implicit deny.

Condition v1 supports `StringEquals`, `BoolEquals`, `NumericEquals`, `DateBefore`, and `DateAfter`. Conditions in one grant use AND; values inside one condition use OR. Missing attributes, type mismatches, malformed conditions, and unknown operators do not match. HTTP clients never provide authorization context; authentication middleware builds the Principal and domain adapters build facts/attributes.

Decision logs include principal/action/resource/effect/reason and matched grant IDs. They exclude JWTs, raw conditions, and domain-fact payloads.

`Principal.GlobalPermissions`, the source of the boundary permission checked in decision step 2, is built from the access token's claims at token issuance (`internal/shared/middleware/auth_jwt.go`), not a live database read. Revoking a role or a global permission from a user therefore has no effect on that user's authorization decisions until their access token expires or is refreshed. This is different from resource scoped grant revocation, which `Authorizer` and `GrantService` both read fresh from `authorization_grants` on every request, so a grant revoked mid-session denies immediately while a global permission revoked mid-session does not.

Effective-action projections use current RBAC assignments from the database, not membership or grant rows alone. A batch resolves role plus direct permissions once, reads active grants once, and applies the same boundary, provider, condition, expiry, explicit-DENY, built-in-ALLOW, scoped-ALLOW, and implicit-DENY rules as mutation enforcement. Any principal-resolution, provider, repository, or condition-evaluation error fails the whole projection; responses never fall back to stale or partially projected actions.

## Resource-scoped role gate

`authorization_role_actions (role_name, resource_type, action_name)` and `authorization_role_bindings (id, principal_user_id, role_name, resource_type, resource_id, granted_by_user_id, created_at, revoked_at)` let a principal hold a named role on one specific resource instance, in addition to (not instead of) direct per-action grants. `authorization_role_actions` is the role definition — which actions a role name currently covers for a resource type, enforced by a foreign key into `authorization_actions` so a role can never reference an action that is not already registered. `authorization_role_bindings` is the assignment — who holds which role on which resource instance; it does not store which actions that entails.

`GormGrantRepository.ListActive` resolves an active role binding into one synthesized `ALLOW` grant per action the binding's role currently covers, by joining the two tables on `(role_name, resource_type)`, and merges those with directly stored grants before returning. A role-expanded entry goes through the exact same decision path as a direct grant (decision steps 1-6 above apply identically), is always `ALLOW`, and carries no `Conditions`/`ValidFrom`/`ExpiresAt` (a role binding is only ever active or revoked). Redefining a role's action set — adding or removing a row in `authorization_role_actions` — takes effect immediately for every existing holder of that role, on every resource instance, with no backfill: bindings reference the role by name and never store a copy of its action set. An explicit `DENY` grant for a specific principal/action/resource still wins over a role-expanded `ALLOW` for that same tuple, exactly as it would over a direct `ALLOW` grant, so a per-principal exception to what a role normally grants does not require any change to the role's definition.

This capability ships no management API: nothing in `internal/authorization`, `GrantService`, or any HTTP route creates, updates, or revokes a role binding or a role's action association. Population of these two tables is out of scope until a later change adds that governance.

## Registered providers

None yet. At startup, `internal/server/wire.go` calls `wireAuthorization(db)` with an empty provider list only to synchronize the empty action catalog and surface initialization errors. It does not construct or expose an unused runtime `Authorizer`, `GrantService`, or `EffectiveActionProjector` bundle. The four authorization tables therefore remain dormant until a resource type registers a `PolicyProvider` and a real consumer is wired. Course does not use this module: Course collaborator authorization is the role-based check on `course_collaborators.role` in `internal/course/infra`, independent of this package. A future change may register Course (or another resource type) as a provider.
