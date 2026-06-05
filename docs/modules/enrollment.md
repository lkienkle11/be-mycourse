# Enrollment Module

_Last audited: 2026-06-04._

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

## Version-switch behavior

- learners always study the currently approved course version
- when a new version is approved, `courses.current_published_version_id` changes and learners move to that version
- progress is preserved for sections / lessons / sub-lessons that keep the same `stable_id`
- removed content no longer contributes to current completion, but historical progress rows remain stored

## Scope note

Payment and commercial enrollment flows are still outside this implementation. The current learner enrollment support is the content-access and progress layer required by the versioned course workflow.
