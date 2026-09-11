## Context

See proposal.md - Why. Everything this change touches is uncommitted (`git status` shows the whole `internal/authorization/` tree as untracked and every Course file below as locally modified, none of it landed on `master`), so "revert" here means "make the working tree match `master` for Course," not a database rollback or a change to shipped behavior.

Confirmed by diffing each modified Course file against `master`:
- `internal/course/domain/errors.go`, `internal/course/application/service_collaborators_bulk.go` (+test), `internal/course/infra/repo_collaborators_bulk.go` (+test), `internal/course/delivery/handler_base.go`: every added line (four new `ErrCourseInvalidCollaborator*` / `ErrCourseOwnerCannotBeManagedAsCollaborator` errors, the `Actions` validation and bulk-plan branching) exists only to support the fine-grained action grant feature and is used nowhere else. A whole-file revert to `master` removes exactly this feature, nothing else.
- `internal/course/infra/repo_admin.go`, `repo_instructor.go`, `repo_access.go`, `repo_collaborators.go`, `repos.go`, `internal/server/wire_course.go`: every addition is either a call into `internal/authorization` (`r.grants.RevokeResource`, `r.grants.RevokePrincipalResource`, `r.authorizer.Authorize` via `ensureEditableDraftForAction`/`requireCourseAction`) or plumbing to reach it (`GormRepository.authorizer`/`grants`/`projector` fields, `NewGormRepository`'s extra parameters). Same conclusion.
- `internal/course/delivery/dto.go`: mixes the `Actions *[]string` field (revert) with a `max=100` bound and a new `NormalizeForValidation` trim/dedupe hook on `addCollaboratorsBulkRequest.UserIDs` (calls the pre-existing, unmodified `sharedutils.PrepareBulkUserIDs`, already used elsewhere in `master`). These two are independent improvements bundled into the same file edit. See Decisions below for how this is handled.

## Goals / Non-Goals

**Goals:**
- After this change, `git diff master -- internal/course/ internal/server/wire.go internal/server/wire_course.go` is empty except for the one intentionally-kept dto.go hunk (see Decisions), and `git status` shows the five new Course authorization files as removed, not modified.
- `internal/authorization/**` and its generic supporting code (RBAC batch lookup, `requestprincipal`, `gormx.WithTransaction`, `auth_jwt.go`'s `GlobalPermissions`) are untouched: same files, same content, same passing tests.
- The server still builds and boots with `internal/authorization` wired but inert (zero `PolicyProvider`s registered).

**Non-Goals:**
- Not redesigning how Course will eventually plug into `internal/authorization`. That is explicitly deferred to a future change, per the user's decision to decouple now and integrate later.
- Not touching `internal/rbac`'s batch permission lookup (`PermissionCodesForUsers`) even though its only current caller is the authorization module's principal resolver — it is generic RBAC infrastructure, not Course-specific, and removing it would require also removing its own tests for no benefit.
- Not fixing or improving the pre-existing role-based Course collaborator authorization (the `master` behavior this reverts to). Any complaints about that model are out of scope here.
- Not preserving `internal/course/application/authorization_policy.go`'s `CollaboratorGrantableActions` dedup work (from the now-deleted `fix-course-authorization-findings` change) — the file it deduped into no longer exists after this revert.

## Decisions

**Decision: revert every fully-authorization-only Course file with `git checkout master -- <path>`, not a hand-edited diff.**

For every file in proposal.md's "Impact" list except `dto.go`, the diff against `master` is confirmed (Context, above) to contain nothing but authorization-integration code. A whole-file checkout is mechanically verifiable (`git diff master -- <path>` becomes empty) and removes any risk of a hand-edit leaving a stray unused import, an orphaned helper, or a half-reverted branch.

Alternatives considered:
- *Hand-edit each file to remove only the authorization hunks.* Rejected: strictly more error-prone than a checkout for files with zero unrelated content, and this repo's `golangci-lint`/`check-architecture` gates would need to catch any mistake instead of the revert being trivially correct by construction.

**Decision: for `dto.go`, keep the `max=100` bound and `NormalizeForValidation`/`PrepareBulkUserIDs` hook; only remove the `Actions` field and the `Role` enum narrowing.**

These two hunks are independent: the size cap and dedupe/trim call an already-existing, unmodified shared utility (`sharedutils.PrepareBulkUserIDs`) and protect the bulk endpoint regardless of whether fine-grained actions exist. Losing this hardening as a side effect of an unrelated revert would be a quiet regression. The `Actions *[]string` field and the `Role` `oneof=EDITOR` narrowing (which exists only to make bulk-add-as-owner impossible now that owner authority is a grant concept) are reverted to `master`'s `UserIDs`/`Role oneof=OWNER EDITOR`.

Alternatives considered:
- *Whole-file checkout of `dto.go` too, for consistency with every other file.* Rejected: it silently drops a real, independently-useful validation improvement that has nothing to do with the authorization extension being cut; the proposal's Impact section would then be misleading about what "revert" means for this one file.
- *Keep the `Actions` field but make it a no-op.* Rejected: dead API surface that documents a capability the backend no longer implements; violates the reuse/no-half-implementation rule and would confuse API consumers.

**Decision: `internal/server/wire.go` calls `wireAuthorization(db, resolver)` with zero providers; it still exists as a wired, callable, empty registry.**

Per the user's explicit choice (option "a" from the exploration): the generic engine gets exercised through the real server wiring (registry construction, `SyncActions`, table access) even with no resource type registered yet, rather than being wired nowhere. `NewRegistry()` with zero providers is valid (confirmed in `internal/authorization/application/registry.go`: `addProvider` is simply never called, no error path requires at least one provider), and `GrantService.SyncActions` against an empty action catalog is a no-op upsert.

Alternatives considered:
- *Don't call `wireAuthorization` from `wire.go` at all.* Rejected per explicit user decision; documented here only so the future change that adds the next `PolicyProvider` (Course or otherwise) knows `wireAuthorization` is already live in `Wire()` and just needs a provider added to the call.

**Decision: edit migration `000034` rather than deleting it.**

`CREATE TABLE authorization_actions` / `authorization_grants` and their indexes are the generic engine's persistence and must ship for `internal/authorization`'s own tests and any future provider to work. Only the two `INSERT` statements that seed Course-specific actions and backfill grants from `course_collaborators` are removed. `000034_authorization_base.down.sql` (drops both tables) needs no change since it already drops everything the up-migration creates. `migrations/authorization_base_test.go` currently asserts the backfill `INSERT INTO authorization_grants` block exists with specific index/uniqueness properties (`TestAuthorizationBaseGrantIndexesAreAdditive`); with the backfill removed, that assertion no longer has a subject and must be rewritten to check the table/index DDL directly instead of the backfill semantics.

Alternatives considered:
- *Delete migration `000034` entirely and let the next change (whenever Course or another resource type is re-added) introduce the tables fresh.* Rejected: the generic engine's own Go tests and package need the tables to exist to be exercised end-to-end (`gorm_repository_test.go`), and re-numbering migrations after this one (if any exist locally) is unnecessary churn for tables that are staying.

**Decision: trim `docs/modules/authorization.md`'s "Course adapter" section rather than deleting the whole doc.**

The rest of the document (Registry/GormGrantRepository/Authorizer/EffectiveActionProjector/GrantService overview, grant statement semantics, decision order, JWT staleness note) describes the generic engine accurately and independent of Course. Only the "Course adapter" section, which describes a provider that no longer exists after this change, is removed.

## Risks / Trade-offs

- [Risk] `dto.go`'s partial (hunk-level) revert is the one place in this change that isn't a mechanical whole-file checkout, so it is the most likely spot for a mistake (e.g. accidentally leaving `Actions` in the request struct). → Mitigation: after editing, `git diff master -- internal/course/delivery/dto.go` must show only the `max=100`/`NormalizeForValidation` hunk as a difference from `master`; task list includes this exact check.
- [Risk] Deleting `internal/course/application/authorization_policy.go` and `internal/course/infra/authorization.go` removes the only current usage of several `internal/authorization` package exports (`PolicyProvider`, `PrincipalFromContext`, etc. via the Course adapter); if any of those exports have no other caller and no test, `golangci-lint`'s unused-export style checks (if configured) could flag them. → Mitigation: run `make check-all` after the revert; `internal/authorization`'s own test suite already exercises the package directly, independent of Course, so this is expected to pass without changes to `internal/authorization`.
- [Risk] `migrations/authorization_base_test.go`'s rewritten assertions could accidentally become too weak (e.g. stop checking the non-unique index requirement that mattered for concurrent grant replacement) once the backfill-specific assertions are removed. → Mitigation: keep the existing index-existence and no-uniqueness-constraint assertions from the current test verbatim; only remove the backfill-block assertions.
- [Risk] The now-deleted `fix-course-authorization-findings` change's one still-relevant piece of code, `GrantService.requireBoundaryPermissions` in `internal/authorization/application/grant_service.go`, stays in the codebase without an open change document describing why it exists. → Mitigation: this is pre-existing, already-implemented, generic code with its own tests (per that change's tasks.md, item 4.3); it needs no further documentation to be correct, and the next change that adds a real `PolicyProvider` will exercise it end-to-end.

## Migration Plan

No database migration is being rolled back (migration `000034` has never been run against any real database; it is edited in place, not superseded). No feature flag, no deploy sequencing:

1. Delete the five Course-only authorization files.
2. Revert the fully-authorization-only Course files and `wire_course.go` with `git checkout master -- <path>` (list in tasks.md).
3. Hand-edit `dto.go` to drop only the `Actions` field and the `Role` enum narrowing, keeping the `max=100`/`NormalizeForValidation` hunk.
4. Hand-edit `internal/server/wire.go` to call `wireAuthorization(db, resolver)` with no provider argument.
5. Hand-edit migration `000034` to drop the two Course-specific `INSERT` blocks; rewrite `migrations/authorization_base_test.go`'s backfill-specific assertions.
6. Trim `docs/modules/authorization.md`'s "Course adapter" section.
7. Run `go build ./...`, `go vet ./...`, then `make check-all`.

Rollback is a plain revert of this change's own commit; nothing persisted or migrated depends on it.
