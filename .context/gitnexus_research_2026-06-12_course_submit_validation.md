# GitNexus Research Note — Course Submit Validation (BE)

Date: 2026-06-12
Repository: `be-mycourse`
Scope: Phase 1 discovery only (no code changes)

## Discovery checklist (temporary-docs/tieu-chuan-check-be-fe)

- Read context summary baseline:
  - `.context/session_summary_2026-06-10_course_validation.md`
- Read docs + module references:
  - `docs/modules/course.md`
  - `docs/sequence_diagrams.md`
  - `temporary-docs/chuc-nang-course-da-lam/be-chuc-nang-course.md`
- Git baseline:
  - reviewed `git log --oneline -20`
  - reviewed `git diff origin/master...HEAD -- internal/course/`
- GitNexus baseline:
  - read `gitnexus://repo/be-mycourse/context`
  - ran `query`, `context`, `impact` for required symbols
  - ran `npx gitnexus analyze --force` (meta embeddings=0, no `--embeddings`)
  - note: MCP context still reports stale; direct tool results were used as primary graph source

## Symbols analyzed

- `SubmitForReview` (`internal/course/infra/repo_review.go`, `internal/course/application/service.go`)
- `updateDraftStatus` (`internal/course/infra/repo_versioning.go`)
- `validateSubLessonPayload` (`internal/course/infra/repo_versioning.go`)
- `validateVersionRefs` (`internal/course/infra/repo_versioning.go`)
- `loadOutline` (`internal/course/infra/repo_access.go`)
- `loadCollaborators` (`internal/course/infra/repo_access.go`)
- `checkUserAccessible` (`internal/auth/application/service_access.go`)

## Impact summary (upstream)

- `updateDraftStatus`: LOW (graph direct callers unresolved in index; source shows review flow callers)
- `validateSubLessonPayload`: LOW; d=1 callers: `CreateSubLesson`, `UpdateSubLesson`
- `validateVersionRefs`: LOW; d=1 caller: `UpdateBasicInfo`
- `loadOutline`: HIGH; shared across detail/load/reorder/delete/review paths
- `loadCollaborators`: HIGH; shared in detail/collaborator CRUD paths
- `checkUserAccessible`: CRITICAL; auth/session/middleware blast radius

## Mandatory risk warning before edits

- HIGH/CRITICAL symbols (`loadOutline`, `loadCollaborators`, `checkUserAccessible`) must be changed with minimal behavior delta.
- For submit validation implementation, prefer call-site reuse and new helper functions over changing these core shared loaders/checkers broadly.

## Reuse vs extend decision

Reuse:
- `validateVersionRefs` for draft basic info reference validation
- `validateSubLessonPayload` logic (extract shared content checks, do not duplicate)
- `loadOutline` for draft tree traversal (call-only, avoid changing existing behavior)

Extend:
- `updateDraftStatus` to run submit-readiness guard before `DRAFT -> IN_REVIEW` transition when `setSubmitted=true`
- `SubmitForReview` remains thin wrapper (no alternate parallel flow)
- auth accessibility logic extraction into shared package (for collaborator validation reuse)

Do not duplicate:
- no parallel validation pipeline in submit path
- no duplicate user-access logic in course infra

## Source-level submit flow (current gap)

Current flow:
- `POST /courses/:id/submit-review`
- `handler_review.submitForReview` -> `CourseService.SubmitForReview` -> `repo_review.SubmitForReview`
- `repo_versioning.updateDraftStatus` updates status only

Gap:
- no persisted draft completeness checks before status flip
- no outline structure/content validation at submit time
- no collaborator accessibility re-validation at submit time

## Planned touch surface (implementation phases)

- New shared access helper:
  - `internal/shared/useraccess/access.go`
- Auth delegate:
  - `internal/auth/application/service_access.go`
- Submit validation:
  - new `internal/course/infra/repo_submit_validation.go`
  - wire in `internal/course/infra/repo_versioning.go`
- Domain and delivery mapping:
  - `internal/course/domain/errors.go`
  - `internal/course/delivery/handler_base.go`
- Tests:
  - new tests for shared access helper
  - new course infra submit-validation tests
  - extend existing sub-lesson payload tests if needed
