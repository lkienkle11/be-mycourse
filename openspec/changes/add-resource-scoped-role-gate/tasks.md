## 0. Session bootstrap and project understanding (AGENTS.md workflow)

- [x] 0.1 Reused the conversation Session ID (`e56de0bc-d831-4d2b-9f59-b60ad0d0029c`), repaired at the same path when the OS temp file went missing again.
- [x] 0.2 Re-read `.context/session-e56de0bc-...md` for the full five-iteration design history.
- [x] 0.3 Confirmed the on-disk state matched iteration 3 (role-as-action-name, 2 tables, `jsonb_exists` join) before starting — this change reinstates iteration 4's two-table design (`authorization_role_actions`, `authorization_role_bindings`) per the user's explicit "4 bảng đi, trước mắt vậy thử" decision.

## 1. Restore the two role-gate tables

- [x] 1.1 Rewrote `migrations/000034_authorization_base.up.sql`: removed `role_names`/`chk_authorization_actions_role_names`/its GIN index from `authorization_actions` (iteration 3 leftovers); re-added `CREATE TABLE authorization_role_actions` and `CREATE TABLE authorization_role_bindings` + `idx_authorization_role_bindings_decision`, exactly as previously built and tested. `go test ./migrations/...` passed.
- [x] 1.2 Rewrote `migrations/000034_authorization_base.down.sql` back to four `DROP TABLE IF EXISTS` statements (role bindings, role actions, grants, actions).
- [x] 1.3 Re-added `TableAuthorizationRoleActions`/`TableAuthorizationRoleBindings` to `internal/shared/constants/dbschema_name.go`; ran `gofmt -w`.

## 2. Restore the row types and join query

- [x] 2.1 In `internal/authorization/infra/gorm_repository.go`: re-added `roleActionRow`/`roleBindingRow` with their `TableName()` methods; replaced iteration 3's `jsonb_exists(a.role_names, g.action_name)`-based `roleExpandedGrants` (joining `authorization_grants` to `authorization_actions`) with the `(role_name, resource_type)`-keyed join between `authorization_role_bindings` and `authorization_role_actions`. `roleExpandedGrants` no longer takes a `now` parameter (role bindings have no validity window, unlike iteration 3's role-as-grant rows). `go build ./...` succeeded.
- [x] 2.2 `mergeGrants` needed no change — same signature and behavior across every iteration.
- [x] 2.3 Confirmed the `domain.GrantRepository.ListActive` doc comment (generic wording, never named a specific table) still reads correctly; no edit needed.

## 3. Restore tests

- [x] 3.1 Kept the four `mergeGrants` tests unchanged.
- [x] 3.2 Fixed `TestRoleExpandedRowToGrant` and `TestRoleExpandedRowToGrantSupportsMultipleIndependentRoles` (added during iteration 3 to cover the multi-role spec requirement, kept here since the requirement still applies) to use `roleExpandedGrantRow.BindingID` instead of iteration 3's `GrantID` field name. `go test ./internal/authorization/...` passed.
- [x] 3.3 Confirmed `internal/authorization/application/` has zero diff; `grantRow`/`actionRow`/`grantToRow`/`rowToGrant`/`Create`/`Revoke`/`ReplaceMany`/`UpsertActions` all byte-for-byte unchanged from the `revert-course-authorization-extension` baseline.

## 4. Restore documentation

- [x] 4.1 Reverted `docs/modules/authorization.md`'s "Resource-scoped role gate" section and the `GormGrantRepository`/"Registered providers" bullets back to describing the two-table (`authorization_role_actions`/`authorization_role_bindings`) design.
- [x] 4.2 Confirmed no remaining reference to `role_names`/`jsonb_exists`/iteration-3 wording anywhere in the file.

## 5. Backend quality gates

- [x] 5.1 `make check-all` passed on this run: exit 0, 0 lint issues, 0 dupl clone groups, both builds succeeded.
- [x] 5.2 Nothing failed; no fixes needed.
- [x] 5.3 Scoped `git status --short` confirmed exactly the files in proposal.md's Impact section changed; no new migration file.

## 6. Post-implementation review

- [x] 6.1 Confirmed `internal/authorization/application/*.go` zero diff; write-path functions unchanged; no unrelated file touched; no secret introduced; no seed data in the new tables.
- [x] 6.2 Re-read both `CREATE TABLE` blocks end to end: FK/index shapes match `authorization_grants`'s existing conventions exactly, as originally verified when this shape was first built.

## 7. Learning summary and final response

- [x] 7.1 Updated `.context/session-e56de0bc-...md` with the full five-iteration history, including the overreach incident (unilateral revert without explicit confirmation) and its correction.
- [x] 7.2 Reported to the user: the module is back to the four-table design (`authorization_actions`, `authorization_grants`, `authorization_role_actions`, `authorization_role_bindings`), quality gates passed, and the explicit note that a two-table alternative (iteration 3) exists, tested, and documented in the session context file if the trade-off is revisited later.
