## Why

The uncommitted `internal/authorization` work ("aws iam fake": a generic, domain-independent IAM-lite engine with `Principal`/`Action`/`Resource`/`Effect`/`Grant`) is being extended in the same branch to give Course collaborators fine-grained, per-action grants (`course_basic_info:update`, `course_outline:update`) issued through `GrantService`/`CoursePolicyProvider`. That extension has spread into every Course mutation entry point (`repo_access.go`, `repo_admin.go`, `repo_instructor.go`, `repo_outline.go`, `repo_collaborators*.go`, `service_collaborators_bulk.go`) and made the collaborator authorization logic hard to follow, before the generic engine itself has even landed.

None of this is committed yet (`internal/authorization/`, migration `000034`, and every touched Course file are uncommitted on this branch), so this is a scope cut on in-flight work, not a rollback of shipped behavior. The Course integration can be redesigned and re-added later, deliberately, once the generic engine has landed on its own.

## What Changes

- **BREAKING (pre-merge only, nothing shipped)**: Course collaborator authorization reverts to the role-based check that exists on `master` (`course_collaborators.role`, membership via `requireCourseAccess`/`requireEditorAccess`/`requireOwnerAccess`). No per-collaborator, per-action grant editing.
- Remove every Course-specific file added for this integration: `internal/course/application/authorization_policy.go` (+test), `internal/course/infra/authorization.go`, `internal/course/delivery/collaborator_actions_test.go`, `internal/course/infra/repo_collaborator_effective_actions_test.go`, `internal/course/infra/repo_admin_lock_order_test.go`.
- Revert the Course files modified for this integration back to their `master` content: `service_collaborators_bulk.go` (+test), `delivery/dto.go`, `delivery/handler_base.go`, `delivery/handler_instructor.go`, `domain/course.go`, `domain/errors.go`, `infra/repo_access.go`, `infra/repo_admin.go`, `infra/repo_collaborators.go`, `infra/repo_collaborators_bulk.go` (+test), `infra/repo_instructor.go`, `infra/repo_outline.go`, `infra/repos.go`, `server/wire_course.go`.
- Keep `internal/authorization/**` (domain, application, infra) and its own tests exactly as-is: this is the reusable engine, kept unblocked for a future resource type.
- Keep the generic supporting pieces this engine depends on: `internal/shared/requestprincipal/**`, `internal/shared/gormx/transaction_context.go` (+test), the `GlobalPermissions` addition in `internal/shared/middleware/auth_jwt.go` (+test), the RBAC batch permission lookup (`internal/rbac/application/service.go`, `domain/repository.go`, `infra/repos.go`, `infra/sql_templates.go` + new batch tests), and the two new table constants in `internal/shared/constants/dbschema_name.go`.
- Rewire `internal/server/wire.go` to call `wireAuthorization(db, resolver)` with zero `PolicyProvider`s (no `courseapp.NewCoursePolicyProvider()` argument), and revert `wire_course.go`'s `wireCourse` signature back to `wireCourse(db)` / `NewGormRepository(db)`.
- Edit migration `000034_authorization_base.up.sql` to drop the two `INSERT` blocks that seed `course_basic_info:update`/`course_outline:update` actions and backfill grants from `course_collaborators`; keep only the `CREATE TABLE authorization_actions`, `CREATE TABLE authorization_grants`, and their indexes. Update `000034_authorization_base.down.sql` and `migrations/authorization_base_test.go` to match (the test currently asserts the backfill `INSERT` exists and must not use `ON CONFLICT DO NOTHING`; without a backfill, that assertion is removed).
- Trim `docs/modules/authorization.md`'s "Course adapter" section, since Course is no longer a registered provider; keep the generic sections (grant statement semantics, decision order, JWT staleness note).
- No spec-level behavior changes: nothing in `openspec/specs/` describes either the fine-grained grant behavior or the role-based behavior it reverts to, so this change sets `skip_specs: true`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None — `openspec/specs/` has no existing capability for Course authorization or the generic engine to modify; this is a pre-merge scope cut, not a spec-level behavior change to a shipped capability.

## Impact

- `internal/course/**`: 14 modified files reverted to `master`, 5 new files deleted (see "What Changes" for the full list).
- `internal/server/wire.go`, `internal/server/wire_course.go`: revert to no-provider / no-authorization wiring for Course.
- `internal/authorization/**`, `internal/shared/requestprincipal/**`, `internal/shared/gormx/transaction_context.go`, `internal/shared/middleware/auth_jwt.go`, `internal/rbac/**` batch lookup: unchanged, kept as the standalone engine.
- `migrations/000034_authorization_base.up.sql`, `.down.sql`, `migrations/authorization_base_test.go`: edited to remove the Course-specific seed data.
- `docs/modules/authorization.md`: "Course adapter" section removed/trimmed.
- Superseded planning artifact: `openspec/changes/fix-course-authorization-findings/` (patched boundary-permission gaps in the exact Course/`GrantService` wiring this change removes) was already deleted from the working tree as part of this decision; its still-relevant piece, the generic `requireBoundaryPermissions` check in `GrantService.requireManager`, is kept because it does not depend on any registered provider.
- Tests: delete the Course-specific authorization tests listed above; `internal/course/**` existing test suites must pass against the reverted (role-based) behavior; `internal/authorization/**` test suites must keep passing unchanged; `go build ./...`, `go vet ./...`, and `make check-all` must pass.
