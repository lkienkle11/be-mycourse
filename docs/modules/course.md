# Course Module

_Last audited: 2026-06-12 (submit-for-review validation, shared useraccess helper, quiz preview enforcement)._

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
  - `slug` (derived from `title` via `utils.SlugifyName` on create and when `title` changes on `PATCH /basic-info`; not accepted from clients; globally unique among active rows — colliding base slugs get `-2`, `-3`, … suffixes)
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
  - mutable versioned rows carry `row_version` (starts at `1` on create — GORM must set `RowVersion: 1` explicitly because zero-value inserts override the column `DEFAULT 1`)
  - `PATCH /basic-info` requires `expected_row_version >= 1` and increments `row_version` on success; accepts `title` (server recomputes `courses.slug` with the same uniqueness rules as create)
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
- reordering is version-local and atomic; `reorderStableIDRows` applies a two-phase `order_index` update (temporary negative indices, then `0..n-1`) so partial reorders never violate `uix_*_order_per_*_active` unique indexes

Sub-lesson content types:

- `VIDEO` → `course_sub_lesson_videos`
- `QUIZ` → `course_sub_lesson_quizzes` + `course_sub_lesson_quiz_options`
- `TEXT` → `course_sub_lesson_texts`

Text lesson content is stored as Quill Delta JSON text, not HTML.

### Field validation (delivery DTOs)

Shared validators in `internal/shared/validate` (`nonwhitespace_min`, `delta_nonwhitespace_min`) use `internal/shared/utils/text_rules.go`.

| Endpoint / entity | Rules |
|-------------------|-------|
| Create course | `title` ≥5 non-whitespace |
| Basic info PATCH | title ≥5; short_description ≥20; about_course Delta ≥30; thumbnail, level, topic UUID required; tag_ids/skill_ids min 1; outcome_ids len 1; preview_video optional |
| Section | title ≥5; description Delta JSON ≥20 non-whitespace text (legacy plain text accepted on read/validate) |
| Lesson | title ≥5; summary Delta JSON ≥20 non-whitespace text (legacy plain text accepted on read/validate) |
| Sub-lesson | title ≥5; `is_preview` allowed only for `VIDEO` and `TEXT` (QUIZ → `ErrCoursePreviewNotAllowedForQuiz`; learner preview outline filters QUIZ) |

## Submit-for-review validation

`POST /api/v1/courses/:courseId/submit-review` triggers `validateDraftForReview` before changing the version status to `IN_REVIEW`. The validation runs inside the same DB transaction and returns `400 Bad Request` if any rule fails.

### Rules checked by `validateDraftForReview`

| Area | Rule | Domain error |
|------|------|-------------|
| Basic info | `title` ≥5, `short_description` ≥20, `about_course` Delta ≥30 non-whitespace chars | `ErrCourseSubmitBasicInfoIncomplete` |
| Basic info | `thumbnail_file_id`, `course_level_id`, `course_topic_id` — non-empty UUID, existing and not-deleted | `ErrCourseSubmitBasicInfoIncomplete` |
| Basic info | `tag_ids` ≥1, `skill_ids` ≥1, `outcome_ids` == 1 | `ErrCourseSubmitBasicInfoIncomplete` |
| Outline | at least 1 section | `ErrCourseSubmitOutlineIncomplete` |
| Outline | each section has at least 1 lesson | `ErrCourseSubmitOutlineIncomplete` |
| Outline | each lesson has at least 1 sub-lesson | `ErrCourseSubmitOutlineIncomplete` |
| Sub-lesson | `VIDEO` — `media_file_id` non-empty, existing, `READY` status | `ErrCourseSubmitInvalidSubLesson` |
| Sub-lesson | `TEXT` — Delta JSON non-whitespace chars ≥1 | `ErrCourseSubmitInvalidSubLesson` |
| Sub-lesson | `QUIZ` — prompt non-empty, ≥1 option, ≥1 correct answer, all option bodies non-empty | `ErrCourseSubmitInvalidSubLesson` |
| Sub-lesson | `QUIZ` with `is_preview = true` — blocked | `ErrCourseSubmitInvalidSubLesson` (wraps `ErrCoursePreviewNotAllowedForQuiz`) |
| Collaborators | at least 1 collaborator | `ErrCourseSubmitCollaboratorRequired` |
| Collaborators | every collaborator must be an active (non-deleted, non-disabled, non-banned) instructor | `ErrCourseCollaboratorInactive` |

All submit domain errors are mapped to HTTP `400` in `mapCourseError` (`delivery/handler_base.go`).

### Shared user access helper

`internal/shared/useraccess` exposes `CheckAccessible(snapshot *Snapshot, now int64) error` and is used by both the submit collaborator validator and the auth service access checks. This avoids duplicated accessibility logic across modules.

## Learner model

There is no separate `internal/enrollment/` package. Learner enrollment and progress currently live inside `internal/course/`.

- `course_enrollments` stores learner-course membership and `current_version_id`
- `course_progress_items` stores progress keyed by `stable_content_id`
- when a new course version is approved, learners move to the new published version and progress is preserved for content that still shares the same stable ids

## HTTP routes

Routes are registered from `internal/course/delivery/routes.go` through `internal/server/router.go`.

Instructor / collaborator routes:

- `GET /api/v1/courses/my`
- `POST /api/v1/courses` — body `{ "title" }` only (`nonwhitespace_min=5`, max 255); slug is computed server-side from `title` via `SlugifyName`. When the base slug is already used by another active course, the server allocates the next free variant (`base`, then `base-2`, `base-3`, …) so each active course has a globally unique slug (`uix_courses_slug_active`).
- `PATCH /api/v1/courses/:courseId/basic-info` — all listed fields required on save except `preview_video_file_id` (optional UUID): `title` (≥5 non-whitespace, server slugify), `short_description` (≥20), `about_course` (Delta JSON, ≥30 non-whitespace text), `thumbnail_file_id`, `course_level_id`, `course_topic_id`, `tag_ids` (≥1), `skill_ids` (≥1), `outcome_ids` (exactly 1), `expected_row_version`.
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

Repo-wide validation passed during the latest audit (2026-06-12):

- `golangci-lint run` — 0 issues
- `go test ./...` — PASS
- `go build ./...` — PASS
- `make check-architecture` — OK
- `make check-dupl` — OK
- `make check-layout` — OK

## Create course transaction

`POST /api/v1/courses` persists everything in one DB transaction (`GormRepository.CreateCourse`):

1. Insert `courses` (`owner_user_id`, server-computed `slug`, UUID v7 `id` via `gormx.EnsureStringID` before GORM `Create`)
2. Insert initial `course_versions` row (`version_no = 1`, `status = DRAFT`, trimmed `title`)
3. Set `courses.current_draft_version_id`
4. Insert `course_collaborators` row (`role = OWNER`)
5. Reload detail via `loadCourseDetail` → `requireCourseAccess`

Access resolution in step 5 reuses existing helpers (`loadCourse`, `loadActiveRow[collaboratorRow]`) instead of a Raw SQL scan into an embedded `courseRow`. The previous Raw scan could leave `ID = 0` even when SQL returned a row, which surfaced as `404` / `3004` (`course not found`) and rolled back the whole transaction.

List endpoints (`GET /courses/my`, learner catalog, pending reviews) use a flat `courseListScanRow` for GORM `Raw().Scan` — embedded `courseRow` is not populated by GORM for joined list queries; rows are mapped back through `asCourseRow()` → `toCourse()`.

Successful response: HTTP **201**, envelope `data` = `domain.CourseDetail` (course root, `collaborator_role`, empty outline, draft version v1, collaborators list including owner).

## Deferred follow-up

The following repository-internal performance refactors are intentionally deferred to a separate task because they are more invasive than the zero-logic cleanup pass:

- `internal/course/infra/repo_access.go:loadOutline` multi-level N+1 query reduction
- `internal/course/infra/repo_versioning.go` version-clone batching / per-row insert reduction
