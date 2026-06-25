# GitNexus Research — Collaborators Popup + Pagination

**Date:** 2026-06-24  
**Branch (BE/FE):** `chore/require-reading-project-skills` (clean working tree, +1 commit ahead of origin)

## Git baseline

| Repo | Branch | Status |
|------|--------|--------|
| be-mycourse | chore/require-reading-project-skills | clean |
| fe-mycourse | chore/require-reading-project-skills | clean |

## Impact preview (upstream, all LOW)

| Symbol | Repo | Risk | Notes |
|--------|------|------|-------|
| `ListCollaborators` | BE | LOW | 0 direct callers in graph; handler + service only |
| `loadCollaborators` | BE | LOW | Internal; used by course detail + add/remove |
| `CourseCollaboratorsTab` | FE | LOW | editor-page only |
| `handleAddCollaborator` | FE | LOW | editor state hook |
| `listCourseCollaboratorsService` | FE | unused today; sole consumer will be new hook |

## Current vs target

- **BE:** `GET /collaborators` returns full array via `ListCollaborators`; `loadCollaborators` embedded in course detail unchanged.
- **FE:** Tab uses `Select` + `useInstructorRosterList(P41)`; list from `courseDetail.collaborators`.

## Reuse audit

| Need | Reuse |
|------|-------|
| BE paginated handler | `httpx.ListPaginated`, `utils.BaseFilter`, `review-history` pattern |
| BE instructor query | SQL from `instructor/infra/repos.go` `ListRoster` + exclude `course_collaborators` + avatar join like `loadCollaborators` |
| FE list hook | `useApiListQuery` |
| FE pagination UI | `InstructorListPagination`, `buildInstructorPageFooterFromInfo` |
| FE URL sync | `course-editor-review-history-tab` + `instructorCourseEditorTabHref` |
| FE dialog | `Dialog`, `Checkbox`, search toolbar like `media-collection-dialog` |

## Files to touch

**BE:** `handler_instructor.go`, `dto.go`, `routes.go`, `repo_instructor.go`, `domain/course.go`, `application/service.go`, tests, swagger/docs

**FE:** `course-editor-collaborators-tab.tsx`, `course-collaborator-picker-dialog.tsx`, hooks, callers, types, `editor-page.tsx`, `use-course-editor-state.ts`, i18n, docs
