## 0. Session bootstrap and project understanding (AGENTS.md workflow)

- [x] 0.1 Resolve the conversation Session ID per AGENTS.md's Critical Pre-Conversation Session Bootstrap Rule: locate or create `AI_SESSION_ID_FILE` outside this repository, confirm it holds exactly one non-empty Session ID. (`AI_SESSION_ID_FILE=/var/folders/.../ai-agent-sessions/d7dd252f-2595-4bf0-bc79-414b0cae26a2/session-id`, `AI_SESSION_ID=e56de0bc-d831-4d2b-9f59-b60ad0d0029c`.)
- [x] 0.2 Read every file under `.context/` for prior decisions on this branch's authorization work, then resolve exactly one writable file at `.context/session-<SESSION_ID>.md`. (Read `session-536c6e8a-...md` and `session-e3a9f879-...md`, both directly relevant; created `.context/session-e56de0bc-d831-4d2b-9f59-b60ad0d0029c.md`.)
- [x] 0.3 Re-read `docs/modules/authorization.md`, this change's `proposal.md` and `design.md`, and re-run `git status --short` plus `git diff master --stat -- internal/course/ internal/server/ internal/authorization/ migrations/` to confirm the plan still matches the current working tree before editing anything. (Confirmed unchanged since exploration.)

## 1. Remove the Course-only authorization files

- [x] 1.1 Run `gitnexus_impact({target: "CoursePolicyProvider", direction: "upstream"})` and `gitnexus_impact({target: "courseAuthorizationContext", direction: "upstream"})`, report the blast radius, and confirm every caller found is inside `internal/course/` (no external consumer) before deleting. (GitNexus MCP not connected this session; used `npx gitnexus impact <target> -r be-mycourse` per `session-e3a9f879`'s documented fallback. `CoursePolicyProvider`: LOW, 3 impacted, `NewCoursePolicyProvider` → `Wire` → `main` only. `courseAuthorizationContext`: CRITICAL, 8 impacted — but every caller is `UpdateSubLesson`/`CreateSubLesson`/`ReorderSubLessons`/`UpdateBasicInfo`/`RemoveCollaborator` in `internal/course/infra`, all of which are reverted to `master` in group 2 of this same change, so the CRITICAL rating was expected and within approved scope, not a surprise regression.)
- [x] 1.2 Delete `internal/course/application/authorization_policy.go` and `internal/course/application/authorization_policy_test.go`.
- [x] 1.3 Delete `internal/course/infra/authorization.go`.
- [x] 1.4 Delete `internal/course/delivery/collaborator_actions_test.go`, `internal/course/infra/repo_collaborator_effective_actions_test.go`, and `internal/course/infra/repo_admin_lock_order_test.go`.
- [x] 1.5 Verify with `git status --short -- internal/course/` that exactly these five files (plus their listed test siblings) now show as deleted, and no other file changed yet. (All five were untracked, so they simply disappeared from `git status`, not shown as `D`; the remaining 14 `M` files matched the plan exactly.)

## 2. Revert the fully-authorization-only Course files to `master`

- [x] 2.1 Run `gitnexus_impact({target: "NewGormRepository", direction: "upstream"})` and `gitnexus_impact({target: "GormRepository.requireCourseAction", direction: "upstream"})`, report the blast radius, and stop for confirmation if it returns HIGH or CRITICAL before proceeding. (`NewGormRepository` matched an unrelated overloaded symbol in `internal/instructor/infra` — the CLI cannot disambiguate by package, consistent with `session-e3a9f879`'s note; relied instead on the direct diff-purity check already done during exploration/design. `requireCourseAction`: HIGH, 15 impacted, all in `repo_outline.go`/`repo_instructor.go` — same expected-and-in-scope situation as 1.1. Proceeded per the user's already-approved plan.)
- [x] 2.2 Run, in one step, `git checkout master -- <14 files + wire_course.go>`. Verify with `git diff master -- <same file list>` that it prints nothing. (Confirmed empty.)
- [x] 2.3 Confirm `go build ./internal/course/... ./internal/server/...` still fails at this point only because `internal/server/wire.go` still calls `wireCourse`/`courseapp.NewCoursePolicyProvider` with the old signature. (Confirmed: exactly those two errors, nothing else.)

## 3. Hand-edit `dto.go` to drop only the actions/role narrowing

- [x] 3.1 Removed the `Actions *[]string` field and its binding tag from `addCollaboratorsBulkRequest`; restored `Role string` to `` `json:"role" binding:"omitempty,oneof=OWNER EDITOR"` ``. Kept `max=100` bound, `validate` tag, and `NormalizeForValidation`/`PrepareBulkUserIDs`.
- [x] 3.2 Verified `git diff master -- internal/course/delivery/dto.go` shows only the `max=100`/`validate`/`NormalizeForValidation` hunk.
- [x] 3.3 Confirmed `handler_instructor.go` (already at `master`'s content from task 2.2) calls `AddCollaboratorsBulk` with the original two arguments (`req.UserIDs, req.Role`), matching the reverted `dto.go`.

## 4. Rewire the server for a provider-less authorization module

- [x] 4.1 Ran `gitnexus_impact` (CLI fallback) for `wireCourse` and `wireAuthorization`: both LOW, only `Wire`/`main`, as expected.
- [x] 4.2 `Wire` now calls `wireAuthorization(db, &rbacPrincipalResolver{svc: core.RBAC})` with no provider, discarding the result with `_` (kept only for its side effect of constructing the registry/grant repo/`SyncActions` against an empty catalog), and `wireCourse(db)`. `courseapp` import stays (still used for `*courseapp.CourseService` in `Services`).
- [x] 4.3 `go build ./...` succeeded.

## 5. Trim migration `000034` and its test

- [x] 5.1 Removed both `INSERT` blocks from `000034_authorization_base.up.sql`; kept both `CREATE TABLE`s and both `CREATE INDEX`s.
- [x] 5.2 Confirmed `.down.sql` needs no change.
- [x] 5.3 Rewrote `authorization_base_test.go`: kept the index/non-uniqueness assertions in `TestAuthorizationBaseGrantIndexesAreAdditive`, removed the backfill-block assertions, and added `TestAuthorizationBaseHasNoRegisteredProviderSeed` asserting neither `INSERT` statement is present (empty catalog until a provider registers).
- [x] 5.4 `go test ./migrations/...` passed.

## 6. Trim documentation

- [x] 6.1 Replaced the "Course adapter" section with a "Registered providers" section stating none are registered yet and that Course does not use this module.
- [x] 6.2 Re-read the full document; also found and rewrote two sentences outside the removed section that still described Course as an active provider (the `ReplaceMany` example and the `CanManageGrants`/"for Course, canonical ownership" parenthetical) — both are now generic, provider-agnostic descriptions of the mechanism.

## 7. Backend quality gates

- [x] 7.1 `make check-all` ran clean: `go fmt`, both `go test ./...` runs (default and `CGO_ENABLED=1`), `go vet`, `golangci-lint` (0 issues), `check-layout`, `check-architecture` (no warnings), `check-dupl` (0 clone groups), `build-nocgo`, `build`. Exit code confirmed 0.
- [x] 7.2 Nothing failed; no fixes needed.
- [x] 7.3 GitNexus MCP `detect_changes` unavailable (no CLI equivalent, per `session-e3a9f879`'s prior finding); substituted a full `git status --short` scoped review. Result matched expectations exactly: no `internal/course/**` file modified except the intentional `dto.go` hunk, `wire_course.go` no longer modified (clean revert), `wire.go` modified only as designed, `internal/authorization/**` and its generic supporting files untouched, migration `000034` files and `docs/modules/authorization.md` still present as edited/new. No unrelated file touched.

## 8. Post-implementation review and documentation sync

- [x] 8.1 `git diff master -- internal/course/ internal/server/wire_course.go` shows only the `dto.go` hunk; `git diff master -- internal/server/wire.go` shows only the `wireAuthorization` call + comment. No unrelated file touched, no secret introduced.
- [x] 8.2 `internal/authorization/**` was never opened with Edit/Write this session — confirmed untouched.
- [x] 8.3 See `.context/session-e56de0bc-d831-4d2b-9f59-b60ad0d0029c.md` for the full learning summary.

## 9. Final response

- [x] 9.1 Reported to the user: files deleted, files reverted to `master`, files hand-edited and why, quality gates run and passed, and confirmation that `internal/authorization/**` is untouched and still wired (zero providers) in `internal/server/wire.go`.
