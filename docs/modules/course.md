# Course Module

_Last audited: 2026-06-07 (multi-instructor draft editing, review, publish, learner progress versioning, zero-logic review refactor cleanup)._

## Overview

`internal/course/` is now implemented and owns:

- course roots (`courses`)
- versioned course content (`course_versions`)
- collaborator membership (`course_collaborators`)
- version-scoped outline data (`course_sections`, `course_lessons`, `course_sub_lessons`)
- sub-lesson detail rows for `VIDEO`, `QUIZ`, and `TEXT`
- edit leases (`course_edit_leases`)
- learner enrollment and progress (`course_enrollments`, `course_progress_items`)

The module follows the standard DDD repo layout:

```text
internal/course/
├── application/
├── delivery/
├── domain/
└── infra/
```

Migration: `migrations/000016_course_management.{up,down}.sql`

## Core model

- `courses` is the stable root record and stores:
  - `owner_user_id`
  - `slug` (derived on create from `title` via `utils.SlugifyName`; not accepted from clients)
  - `current_published_version_id`
  - `current_draft_version_id`
- `course_versions` stores the editable and published snapshots:
  - `DRAFT`
  - `IN_REVIEW`
  - `APPROVED`
  - `REJECTED`
- Only one active draft exists per course at a time.
- Learners always read from `current_published_version_id`.
- Instructor edits always write to the draft version, never directly to the published version.

## Collaboration and conflict control

- Collaborator roles:
  - `OWNER` — delete course, manage collaborator membership
  - `EDITOR` — update basic info, outline, submit draft for review
- Optimistic locking:
  - mutable versioned rows carry `row_version`
  - stale saves return a conflict (`ErrCourseOptimisticLock`)
- Resource leases:
  - stored in `course_edit_leases`
  - resource scopes:
    - `OUTLINE_ROOT`
    - `SECTION`
    - `LESSON`
    - `SUB_LESSON`
  - lease acquire / heartbeat / release endpoints exist for instructor UI coordination

## Outline model

- `course_sections` belong to a version
- `course_lessons` belong to a section and a version
- `course_sub_lessons` belong to a lesson and a version
- each outline node has a stable business UUID (`stable_id`) that survives version cloning
- reordering is version-local and atomic

Sub-lesson content types:

- `VIDEO` → `course_sub_lesson_videos`
- `QUIZ` → `course_sub_lesson_quizzes` + `course_sub_lesson_quiz_options`
- `TEXT` → `course_sub_lesson_texts`

Text lesson content is stored as Quill Delta JSON text, not HTML.

## Learner model

There is no separate `internal/enrollment/` package. Learner enrollment and progress currently live inside `internal/course/`.

- `course_enrollments` stores learner-course membership and `current_version_id`
- `course_progress_items` stores progress keyed by `stable_content_id`
- when a new course version is approved, learners move to the new published version and progress is preserved for content that still shares the same stable ids

## HTTP routes

Routes are registered from `internal/course/delivery/routes.go` through `internal/server/router.go`.

Instructor / collaborator routes:

- `GET /api/v1/courses/my`
- `POST /api/v1/courses` — body `{ "title" }` only; slug is computed server-side (rejected only when slugify yields empty)
- `GET /api/v1/courses/:courseId`
- `POST /api/v1/courses/:courseId/draft/prepare`
- `PATCH /api/v1/courses/:courseId/basic-info`
- `DELETE /api/v1/courses/:courseId`
- collaborator CRUD under `/api/v1/courses/:courseId/collaborators`
- outline CRUD / reorder under `/api/v1/courses/:courseId/{sections,lessons,sub-lessons,...}`
- lease routes under `/api/v1/courses/:courseId/leases/*`
- review submission routes:
  - `POST /api/v1/courses/:courseId/submit-review`
  - `POST /api/v1/courses/:courseId/reopen-draft`

Admin / sysadmin review routes:

- `GET /api/v1/course-reviews/pending`
- `POST /api/v1/course-reviews/:courseId/approve`
- `POST /api/v1/course-reviews/:courseId/reject`

Learner routes:

- `GET /api/v1/learner-courses`
- `GET /api/v1/learner-courses/:courseId`
- `POST /api/v1/learner-courses/:courseId/enroll`
- `GET /api/v1/learner-courses/:courseId/progress`
- `POST /api/v1/learner-courses/:courseId/progress`

## Permissions

The module reuses the existing permission catalog:

| Permission | Typical use |
|---|---|
| `course:create` | create course |
| `course:update` | edit draft, outline, collaborators (business rules still restrict owner-only actions) |
| `course:delete` | route-level delete gate; business logic still requires `OWNER` |
| `course:read` | learner course list/detail/progress |
| `course_instructor:read` | instructor editable course list and detail |
| `admin:modify` | approve / reject review queue |

## Wiring

- Service: `internal/course/application/CourseService`
- Repository: `internal/course/infra/GormRepository`
- Handler: `internal/course/delivery/Handler`
- Server wiring:
  - `internal/server/wire_course.go`
  - `internal/server/wire.go`
  - `internal/server/router.go`

## Validation snapshot

Repo-wide validation passed during the latest audit:

- `golangci-lint cache clean`
- `golangci-lint run`
- `go test ./...`
- `go build ./...`
- `make check-architecture`
- `make check-dupl`
- `make check-layout`
- `make build`

## Deferred follow-up

The following repository-internal performance refactors are intentionally deferred to a separate task because they are more invasive than the zero-logic cleanup pass:

- `internal/course/infra/repo_access.go:loadOutline` multi-level N+1 query reduction
- `internal/course/infra/repo_versioning.go` version-clone batching / per-row insert reduction
