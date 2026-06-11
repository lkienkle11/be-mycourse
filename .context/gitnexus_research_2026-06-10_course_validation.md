# GitNexus Research — Course Field Validation (2026-06-10)

## Symbols reuse
- `courseTitleAndSlug` — extend with `CountNonWhitespace` min 5
- `validate.BindJSON` + `validate.Struct` — new tags `nonwhitespace_min`, `delta_nonwhitespace_min`
- `CourseDeltaEditor` (FE) — reuse for about_course; no react-quill

## Symbols change + d=1 callers
| Symbol | Risk | d=1 |
|--------|------|-----|
| `courseTitleAndSlug` | LOW | `CreateCourse`, `UpdateBasicInfo` |
| `updateBasicInfoRequest` DTO | LOW | `updateBasicInfo` handler |
| `validateSubLessonPayload` | LOW | `CreateSubLesson`, `UpdateSubLesson` |
| `filterPreviewOutline` | LOW | learner outline path in `repo_access` |

## New symbols
- `utils.CountNonWhitespace`, `utils.CountDeltaNonWhitespace`
- `ErrCourseTitleTooShort`, `ErrCoursePreviewNotAllowedForQuiz`

## Processes
- Create course → DTO validate → `courseTitleAndSlug` → repo
- PATCH basic-info → DTO validate (all required) → handler trim → `UpdateBasicInfo` → slug from title
- Sub-lesson upsert → DTO validate → `validateSubLessonPayload` (QUIZ+preview rejected)
