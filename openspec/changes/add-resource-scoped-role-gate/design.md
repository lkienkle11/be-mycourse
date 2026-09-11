## Context

See proposal.md - Why, including the full five-iteration history. Relevant current-state facts:

- `domain.GrantRepository.ListActive(ctx, GrantQuery, now) ([]Grant, error)` is the single read path both `Authorizer.authorize` (`internal/authorization/application/authorizer.go:42`) and `EffectiveActionProjector.Project` (`internal/authorization/application/effective_action_projector.go:70`) call to obtain the candidate grants for a decision. Neither caller does anything with the returned `[]domain.Grant` beyond what `matchingGrantIDs`/`decidePreparedAuthorization` already do. Both call sites are indifferent to *where* a `Grant` value came from.
- `authorization_actions (action_name, resource_type, boundary_permission, description, created_at, updated_at)`, `PRIMARY KEY (action_name, resource_type)`, is the existing action catalog. `authorization_grants` already has `CONSTRAINT fk_authorization_grants_action FOREIGN KEY (action_name, resource_type) REFERENCES authorization_actions (action_name, resource_type)` — this exact composite FK shape is reused by the new `authorization_role_actions` table below, from a second source, unchanged in the target table.
- `migrations/000034_authorization_base.up.sql`/`.down.sql` have never run against any database. Editing them in place, rather than adding a new migration, carries no compatibility risk.
- No live PostgreSQL test harness exists in this environment. `internal/authorization/infra/gorm_repository_test.go` therefore only unit-tests pure helper functions, never the GORM query construction itself. This design follows the same pattern.
- A prior iteration (see proposal.md's history, iteration 3) stored a role name directly as an `authorization_grants.action_name` value with no schema change at all, achieving a two-table total. It was built and verified working, then set aside in favor of this shape: the user weighed "fewer tables, but a column whose meaning depends on an unwritten convention" against "more tables, but every column has exactly one meaning" and chose the latter for now.

## Goals / Non-Goals

**Goals:**
- Every table and column has exactly one, schema-visible meaning. `authorization_grants` and `authorization_actions` are not modified at all: no new columns, no relaxed constraints, no new `CHECK` constraints. Every existing row, write path, and test keeps working unchanged.
- `Authorizer` and `EffectiveActionProjector` see role-expanded access exactly like a direct grant, with literally zero code changes to `internal/authorization/application/*.go`.
- Redefining a role's action set (insert/delete a row in `authorization_role_actions`) changes every existing holder's effective access on the very next decision — no batch job, no backfill, no cache invalidation to reason about.
- A role can only ever be associated with an action that is already registered in `authorization_actions`, enforced by the database (a foreign key), not by application code alone.
- Works for any resource type without a code change to that resource type's `PolicyProvider` — a role name is pure data, never declared in Go.

**Non-Goals:**
- No service, no HTTP route, no `GrantService` method to create/update/revoke a role binding or a role's action association. This change ships the read-side gate only.
- No `ValidFrom`/`ExpiresAt`/`Conditions` on `authorization_role_bindings`. A role binding is either active or revoked, nothing in between.
- No `DENY` role bindings — a role binding only ever contributes `ALLOW` (there is no `effect` column on `authorization_role_bindings` at all). Per-principal subtraction from a role's normally-granted action is already possible today with a direct `DENY` grant in `authorization_grants`.
- No change to `Registry`, `PolicyProvider`, or any resource type's action declarations.

## Decisions

**Decision: two new, single-purpose tables — `authorization_role_actions` (role definition) and `authorization_role_bindings` (role assignment) — each with a real foreign key into existing schema, rather than reusing or extending `authorization_grants`/`authorization_actions`.**

```sql
CREATE TABLE authorization_role_actions (
    role_name     VARCHAR(100) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    action_name   VARCHAR(100) NOT NULL,
    created_at    BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    PRIMARY KEY (role_name, resource_type, action_name),
    CONSTRAINT fk_authorization_role_actions_action
        FOREIGN KEY (action_name, resource_type)
        REFERENCES authorization_actions (action_name, resource_type)
        ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE authorization_role_bindings (
    id                 UUID PRIMARY KEY,
    principal_user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_name          VARCHAR(100) NOT NULL,
    resource_type      VARCHAR(64) NOT NULL,
    resource_id        VARCHAR(128) NOT NULL,
    granted_by_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at         BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    revoked_at         BIGINT NULL
);

CREATE INDEX idx_authorization_role_bindings_decision
    ON authorization_role_bindings (principal_user_id, resource_type, resource_id)
    WHERE revoked_at IS NULL;
```

"Does role R cover action A for resource type T?" is `SELECT 1 FROM authorization_role_actions WHERE role_name = R AND resource_type = T AND action_name = A` — a plain primary-key lookup. Redefining a role is a single-row `INSERT`/`DELETE` against `authorization_role_actions`, immediately visible to every existing `authorization_role_bindings` row for that role, since bindings reference the role by name and never store a copy of its action set.

Alternatives considered (see proposal.md's full numbered history):
- *Two new tables with no foreign key into the existing catalog* (iteration 1). Rejected: looked like isolated schema.
- *Extending `authorization_grants`/`authorization_actions` in place with nullable columns* (not part of this history's final comparison, tried and reverted earlier). Rejected: dual-purpose rows, hard to key correctly.
- *A role name stored as an `authorization_grants.action_name` value, resolved via a `role_names` JSONB array on `authorization_actions`* (iteration 3). Achieves a two-table total with zero schema changes to either existing table. Built, tested, passing. Set aside: `action_name`'s meaning becomes context-dependent (a concrete action or a role reference) with no schema-level marker distinguishing the two — a real complexity cost, just relocated from "table count" to "value semantics." The user explicitly chose this iteration's shape (more tables, unambiguous column meanings) over that one, to try first.

**Decision: expand roles into synthetic `domain.Grant` values inside `GormGrantRepository.ListActive`, not as a new concept in `domain`/`application`.**

`ListActive` already returns `[]domain.Grant`; both callers already treat every element uniformly. A second query, entirely inside `internal/authorization/infra/gorm_repository.go`, joins `authorization_role_bindings` to `authorization_role_actions` on `(role_name, resource_type)`, filtered by the same `PrincipalUserIDs`/`Resource`/`ActionNames` the caller asked for and `revoked_at IS NULL`, and maps each result to `domain.Grant{Effect: EffectAllow, ID: "role:"+bindingID+":"+actionName, ValidFrom: nil, ExpiresAt: nil, Conditions: nil, RevokedAt: nil}`. The two result slices are combined by a pure, unit-testable merge helper before `ListActive` returns.

Because `authorization_grants`'s schema does not change under this design, `grantRow`, `grantToRow`, and `rowToGrant` need **no changes at all**.

Alternatives considered:
- *Add a `RoleRepository` alongside `GrantRepository` and teach `Authorizer`/`EffectiveActionProjector` to query both.* Rejected: reintroduces a new axis of complexity in the decision engine.
- *Materialize role-expanded rows into `authorization_grants` itself.* Rejected: reintroduces the exact backfill/staleness problem this change exists to remove.

**Decision: the merge is a pure function, unit-tested without a database; the join query itself is not integration-tested (same limitation as the rest of this file).**

```go
func mergeGrants(direct, roleExpanded []domain.Grant) []domain.Grant
```

Simple concatenation plus the same `principal_user_id, action_name, effect, id` ordering already applied to direct grants. No deduplication is needed: `matchingGrantIDs` already tolerates multiple `ALLOW` entries for the same tuple, and a duplicate-looking `ALLOW` from two different sources remains individually traceable via its `ID`.

## Risks / Trade-offs

- [Risk] The join query in the new role-expansion method cannot be integration-tested in this environment (no live Postgres harness). → Mitigation: keep the query as close as possible in shape to the existing, already-reviewed direct-grant query in the same file; push all non-SQL logic into the pure, tested merge helper.
- [Risk] Without any write path in this change, the new tables cannot be exercised by anything other than test fixtures inserting rows directly. → Mitigation: this is the explicit, deliberate scope cut the user asked for.
- [Risk] Two callers now issue two queries per `ListActive` invocation instead of one, even when zero role bindings exist for a resource type. → Mitigation: not a concern yet (no resource type is registered); a cheap short-circuit is a straightforward follow-up if it ever matters.
- [Risk] This design carries two more tables than the alternative that was built and verified working (iteration 3), for a module with zero current consumers. → Mitigation: this trade was made consciously — the user prioritized schema-level clarity (every column has one meaning) over table count, explicitly as something to "try for now"; iteration 3's design remains a documented, working alternative in this session's context file if this one proves harder to live with in practice.

## Migration Plan

No feature flag, no deploy sequencing beyond the standard migration order:

1. Add two `CREATE TABLE` statements + one partial index (no seed data) to `migrations/000034_authorization_base.up.sql`, and the matching `DROP TABLE`s to `.down.sql`.
2. Add the two table-name constants to `internal/shared/constants/dbschema_name.go`.
3. Add the two new row types, the role-expansion query method, and the pure merge helper to `internal/authorization/infra/gorm_repository.go`; update `ListActive` to call both and merge. `grantRow`/`grantToRow`/`rowToGrant` are not touched.
4. Add the doc comment to `domain.GrantRepository.ListActive`.
5. Add unit tests for the merge helper and for the role-expansion row-mapping logic.
6. Update `docs/modules/authorization.md` with a section, explicit about the "no management API yet" boundary.
7. Run `go build ./...`, `go vet ./...`, `go test ./internal/authorization/... ./migrations/...`, then `make check-all`.

Rollback is a plain revert; no data is written by this change outside of what a future change's own migration or seed would add.
