# Course Collaboration Handoff

_Last updated: 2026-06-05_

## Purpose

This document is the current backend checkpoint for the course collaboration and versioned publishing work across `be-mycourse` and `fe-mycourse`.

It summarizes:

- the backend context
- what was implemented
- what was refactored and fixed
- what still remains
- what should be done next

## Backend scope that was targeted

- Multi-instructor course collaboration
- Safe draft editing without affecting the learner-facing published version
- Versioned course publishing with admin/sysadmin approval
- Conflict prevention for course outline editing
- Learner enrollment and version-aware progress tracking
- Stable-content-based progress carry-over across approved versions

## Backend context

The backend now contains a dedicated `course` bounded context under:

```text
internal/course/
├── application/
├── delivery/
├── domain/
└── infra/
```

The feature is built around these core rules:

- `course` is the root entity
- a course has at most one active draft version at a time
- learners only use the currently approved published version
- instructors edit the draft only (`DRAFT` status for mutations; submit keeps `version_no`)
- approval promotes the submitted version to the new live version; next prepare uses `max(version_no)+1`
- reject freezes the submitted row as `REJECTED` and forks a new `DRAFT` at `max(version_no)+1`
- progress is tied to stable content identities instead of version-specific row ids

## Tasks completed

### Core backend feature implementation

Implemented:

- course root and versioned publishing model
- collaborator roles: `OWNER`, `EDITOR`
  - `OWNER` — delete course, manage collaborators, prepare draft, submit for review, reopen rejected draft
  - `EDITOR` — edit basic info and outline only (no prepare/submit/reopen)
- draft states: `DRAFT`, `IN_REVIEW`, `REJECTED`, `APPROVED`
- course basic info editing
- course outline editing:
  - sections
  - lessons
  - sub-lessons
- sub-lesson content types:
  - `VIDEO`
  - `QUIZ`
  - `TEXT`
- course collaborator listing / add / remove
- learner enrollment
- learner progress persistence
- review submit / reopen / approve / reject flow
- lease acquire / heartbeat / release flow
- optimistic locking with `row_version`

### Database work

Migration added:

- `migrations/000016_course_management.up.sql`
- `migrations/000016_course_management.down.sql`

Tables added:

- `courses`
- `course_versions`
- `course_version_tags`
- `course_version_skills`
- `course_version_outcomes`
- `course_collaborators`
- `course_sections`
- `course_lessons`
- `course_sub_lessons`
- `course_sub_lesson_videos`
- `course_sub_lesson_texts`
- `course_sub_lesson_quizzes`
- `course_sub_lesson_quiz_options`
- `course_edit_leases`
- `course_enrollments`
- `course_progress_items`

### API and routing work

Implemented instructor, review, and learner backend endpoints through:

- `internal/course/delivery/`
- `internal/server/wire_course.go`
- updates in `internal/server/wire.go`
- updates in `internal/server/router.go`

### Constants and shared wiring

Updated:

- `internal/shared/constants/dbschema_name.go`

## Important fixes and refactors already completed

### Delivery layer cleanup

The original large course handler file was split into focused files:

- `internal/course/delivery/handler_base.go`
- `internal/course/delivery/handler_instructor.go`
- `internal/course/delivery/handler_outline.go`
- `internal/course/delivery/handler_review.go`
- `internal/course/delivery/handler_learner.go`

This removed repeated route glue and centralized:

- course id parsing
- request binding
- shared response writing
- shared course error mapping

### Repository cleanup

The original monolithic repository implementation was split into focused files:

- `internal/course/infra/repos.go`
- `internal/course/infra/repo_helpers.go`
- `internal/course/infra/repo_instructor.go`
- `internal/course/infra/repo_outline.go`
- `internal/course/infra/repo_review.go`
- `internal/course/infra/repo_learner.go`
- `internal/course/infra/repo_versioning.go`
- `internal/course/infra/repo_access.go`

This refactor removed the main duplication hotspots by:

- extracting shared active-row loaders
- extracting optimistic update helpers
- extracting shared draft-outline delete flows
- splitting clone/version/ref validation logic
- reducing file length to repo-standard ranges

## Validation completed

### Passed

- `golangci-lint run`
- `go test ./...`
- `go build ./...`
- `make check-architecture`
- `make check-dupl`
- `make check-layout`
- `npx gitnexus analyze --force`

### Validation caveat

- `golangci-lint cache clean` hit a local cache-directory cleanup issue on this machine (`directory not empty`), but lint execution itself still completed successfully afterward.

## Docs already updated in backend

Updated earlier during this work:

- `docs/modules/course.md`
- `docs/modules/lesson.md`
- `docs/modules/enrollment.md`
- `docs/modules.md`

## Tasks not completed yet

### Deeper automated coverage

Still needed:

- integration tests for lease contention
- integration tests for optimistic lock conflicts
- approval/version switching tests
- progress carry-over tests using stable content ids

### Product surface still deferred

Still intentionally not implemented:

- pricing backend behavior
- certificate backend behavior
- broader learner course-player UX concerns

## Current backend status

- The backend course feature is functionally implemented.
- The course package is structurally clean, route-wired, lint-clean, and repo-wide validation passed.
- Remaining work is now mostly deeper automated coverage and any future product expansion beyond this phase.

## Recommended next steps

1. Add targeted integration tests for leases, optimistic locking, review approval switching, and stable-id progress migration.

2. Run cross-repo manual QA with the frontend against a real backend dataset, especially multi-instructor outline editing and draft approval handoff.

3. Keep pricing, certificate, and learner-player work in a separate follow-up scope so the current collaboration/versioning behavior stays stable.

4. Update backend `.context` after the code shape is considered stable.

5. Add backend integration tests for the conflict and publishing edge cases before extending the feature further.

## Frontend companion doc

For the frontend checkpoint, see:

- `/Users/kienlt/Documents/projects/mycourse-full/fe-mycourse/docs/course-collaboration-handoff-2026-06-04.md`
