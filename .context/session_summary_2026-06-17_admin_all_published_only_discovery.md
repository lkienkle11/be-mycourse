# Phase 1 — admin all courses shows draft (2026-06-17)

## Bug
All courses page shows "Bản nháp" because SQL uses `COALESCE(dv.*, pv.*)` — draft wins over published.

## Expected
- List only courses with **approved published version** (`pv.status = APPROVED`)
- Display **published version** title + version_no
- No status column on FE all/trash pages
- Remove useless `?approval=` filter (always approved published)

## Files
- BE: `repo_admin.go`, `handler_admin.go`, `domain/course.go`, `service.go`, docs
- FE: `course-admin-list-columns.tsx`, `course-admin-all-page.tsx`, `course-admin-trash-page.tsx`, API callers/hooks, i18n

## Risk: LOW
