# Enrollment Module

_Last audited: 2026-09-27 — added `GET /learner-courses/continue` (`openspec/changes/add-home-catalog-apis`). Prior: 2026-06-07._

There is no standalone `internal/enrollment/` package yet.

Enrollment and learner progress are currently implemented inside `internal/course/` because they are tightly coupled to course version selection and stable-content progress migration.

## Current tables

- `course_enrollments`
  - one learner-course membership row
  - stores `current_version_id`
- `course_progress_items`
  - stores learner progress keyed by `stable_content_id`
  - keeps progress tied to business-stable outline identities instead of version-local row ids

## Current behavior

- learner enrollment is created via:
  - `POST /api/v1/learner-courses/:courseId/enroll`
- learner course detail is read via:
  - `GET /api/v1/learner-courses/:courseId`
- learner progress is read / saved via:
  - `GET /api/v1/learner-courses/:courseId/progress`
  - `POST /api/v1/learner-courses/:courseId/progress`
- the caller's own in-progress courses, ordered by most-recent learning activity, are read via:
  - `GET /api/v1/learner-courses/continue` (`limit`, default 4, max 10) — orders by `MAX(course_progress_items.last_interacted_at)` per enrollment, falling back to `course_enrollments.created_at` when the learner hasn't started any lesson yet; each item includes a completed/total **sub-lesson** count (every kind — `VIDEO`/`QUIZ`/`TEXT` — not video-only, since a course's outline legitimately mixes content kinds), derived from `course_sub_lessons` (the authoritative outline) joined by `stable_id`, not from the client-supplied `course_progress_items.content_type` (see `internal/course/infra/repo_learner.go`'s `continueLearningQuery`)

## Version-switch behavior

- learners always study the currently approved course version
- when a new version is approved, `courses.current_published_version_id` changes and learners move to that version
- progress is preserved for sections / lessons / sub-lessons that keep the same `stable_id`
- removed content no longer contributes to current completion, but historical progress rows remain stored

## Scope note

Payment and commercial enrollment flows are still outside this implementation. The current learner enrollment support is the content-access and progress layer required by the versioned course workflow.
