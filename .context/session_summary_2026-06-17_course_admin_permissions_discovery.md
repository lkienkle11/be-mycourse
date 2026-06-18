# Discovery — course admin granular permissions (2026-06-17)

## Problem
`admin:modify` / `sysadmin:modify` are **shell gate** permissions only — must not guard API routes.

## New permissions (P59–P66)
| ID | Name | Route |
|----|------|-------|
| P59 | `course_review:read` | GET `/course-reviews/pending` |
| P60 | `course_review:approve` | POST `.../approve` |
| P61 | `course_review:reject` | POST `.../reject` |
| P62 | `course_catalog:read` | GET `/course-admin/courses` |
| P63 | `course_catalog:trash` | POST `.../trash` |
| P64 | `course_trash:read` | GET `/course-admin/courses/trash` |
| P65 | `course_trash:restore` | POST `.../restore` |
| P66 | `course_trash:delete` | DELETE `.../permanent` |

Grant P59–P66 to **sysadmin** + **admin**.

## FE menu mapping
- Courses group: any of P62, P59, P64
- All: P62 | Reviewing: P59 | Trash: P64

## Other
- Confirm dialog before trash / permanent delete (reuse `ConfirmDeleteDialog`)
- Preview route: `/reviewing/[courseId]/preview` (no versionId)

## Risk: MEDIUM (RBAC catalog + JWT re-login)
