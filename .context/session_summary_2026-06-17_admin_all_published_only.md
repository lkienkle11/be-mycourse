# Session — admin all courses published-only (2026-06-17)

## Root cause
`COALESCE(dv.*, pv.*)` preferred draft over published → draft-only courses appeared as "Bản nháp".

## BE
- `ListAdminCourses`: INNER JOIN published version, `pv.status = APPROVED` only
- Display fields from `pv` (title, version_no, review_status)
- `draft_review_status` for trash eligibility
- Removed `?approval=` filter and `AdminCourseListFilter`

## FE
- Removed status column + filter dropdown on All courses page
- Version column = published version
- `canMoveCourseToTrash` uses `draft_review_status`
- Review queue still shows status column

## Quality
- `make test-all` PASS
- `npm run check-all` PASS
