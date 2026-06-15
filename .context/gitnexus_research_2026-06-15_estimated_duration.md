# GitNexus research — estimated duration (2026-06-15)

## Symbols to extend (do not recreate)

| Layer | Symbol | File |
|-------|--------|------|
| DB | `course_sub_lessons` | migration `000022` |
| Row | `subLessonRow` | `internal/course/infra/repos.go` |
| Domain | `SubLesson`, `Lesson`, `Section`, `UpsertSubLessonInput` | `internal/course/domain/course.go` |
| DTO | `subLessonRequest` | `internal/course/delivery/dto.go` |
| Handler | `upsertSubLesson` | `internal/course/delivery/handler_outline.go` |
| Read | `loadOutline`, `loadSubLessonDomain`, `toSubLesson` | `repo_access.go`, `repos.go` |
| Write | `CreateSubLesson`, `UpdateSubLesson` | `repo_outline.go` |
| Clone | `cloneSubLessonRows` | `repo_versioning.go` (struct copy auto-includes new column) |
| New | `duration.go` helpers | `internal/course/infra/duration.go` |

## Impact analysis

- `loadOutline` — **HIGH** upstream (6 d=1 callers: `loadCourseDetail`, `loadLearnerCourseDetail`, `validateDraftOutline`, reorder/delete paths). Change is additive (new JSON fields + post-load resolution); no signature break.
- `SubLesson` struct — **LOW** (additive field).
- `CreateSubLesson` / `UpdateSubLesson` — extend write path only.

## Current state

- No `estimated_duration_ms` on `course_sub_lessons`.
- `media_files.duration` is int64 **seconds** (see `internal/media/domain/media.go`).
- `loadSubLessonDomain` joins video URL via `mediaURL` only.
- `git diff main...HEAD -- internal/course/` — no in-flight outline changes on current branch.

## Write rules

- VIDEO: force `estimated_duration_ms = 0` in DB; API resolves from `media_files.duration * 1000`.
- TEXT/QUIZ: persist client ms; validate `0 <= ms <= 999h`.
- Kind → VIDEO: stored ms reset to 0 on update.

## Test targets

- `resolveSubLessonEstimatedDurationMs`, `applyOutlineEstimatedDurations`, `normalizeSubLessonEstimatedDurationMs` in `duration_test.go`.
