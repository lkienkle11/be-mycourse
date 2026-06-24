# GitNexus Research — Course Review Enhancements (BE)

Date: 2026-06-23  
Repository: `be-mycourse`  
Scope: Phase 1 discovery only (no code changes)

## Context baseline

- No prior session on course review history; related: `session_summary_2026-06-17_course_admin_permissions.md`, `docs/modules/course.md` (approve/reject flows, version fork on reject).
- FE already has dropdown row actions (`session_summary_2026-06-23_course_review_actions_dropdown.md` on FE branch).

## Docs gap

| Doc | Gap |
|-----|-----|
| `docs/modules/course.md` | Approve body `{}`; reject `min=1,max=2000`; no `approval_note`; no `GET .../review-history` |
| `docs/api_swagger.yaml` | Same — approve empty body, reject max 2000 |
| `docs/curl_api.md` | No review-history curl; approve/reject examples stale |

## Git baseline

- Branch: `feat/gorm-sql-console-logger`, clean working tree, ahead of origin by 3 commits.
- No in-progress course-review changes on BE.

## GitNexus

- Index: 2 commits stale (acceptable for discovery).
- `query("course review approve reject history")` → `repo_review.go`, routes, `docs/modules/course.md`.
- `context(ApproveDraft)` / `context(RejectDraft)` — ambiguous infra vs application; use infra impl.
- `impact(ApproveDraft)` upstream → **LOW**, 0 direct callers in graph (interface/repo layer).
- `impact(RejectDraft)` — same pattern expected LOW.

## Symbols reuse

- `utils.BaseFilter`, `utils.BuildPage`, `response.OKPaginated` — review-history pagination.
- `nonwhitespace_min` validator — approve/reject DTOs (already used in course DTOs).
- `requireEditorAccess` — history endpoint access check.
- `courseBodyOK` — approve body bind (same as reject).

## Symbols change + d=1 callers

| Symbol | Change | Risk | d=1 |
|--------|--------|------|-----|
| `ApproveDraft` | +`approvalNote string`; persist `approval_note` | LOW | `CourseService.ApproveDraft`, `approveDraft` handler |
| `RejectDraft` | DTO `nonwhitespace_min=5,max=500` only | LOW | handler unchanged signature |
| `Repository` | +`ListReviewHistory` | LOW | service + handler |
| `courseVersionRow` | +`ApprovalNote` field | LOW | mapping in `mapCourseVersionRow` |

## New symbols

- Migration `000026_course_approval_note`
- `approveDraftRequest`, `listReviewHistoryQuery` DTOs
- `CourseReviewHistoryItem`, `ReviewHistoryFilter` domain types
- `ListReviewHistory` repo/service/handler
- Route `GET /courses/:courseId/review-history`

## Phase 2 file list (chốt)

- `migrations/000026_course_approval_note.{up,down}.sql`
- `internal/course/infra/repos.go`, `repo_review.go`
- `internal/course/domain/course.go`
- `internal/course/delivery/dto.go`, `handler_review.go`, `routes.go`
- `internal/course/application/service.go`
