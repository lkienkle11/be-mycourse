# Session — fix ListAdminCourses Raw args (2026-06-17)

## Bug
`GET /api/v1/course-admin/courses` → 500  
GORM: `expected 0 arguments, got 1` at `repo_admin.go`

## Cause
`Raw(q, map[string]any{})` passed even when SQL has no `@` placeholders (`ApprovedOnly=false`).

## Fix
`Raw(q)` when no filters; `Raw(q, map)` only when `ApprovedOnly`.

## Quality
- `make test-all` PASS

## Manual
Restart BE (`CGO_ENABLED=1 MIGRATE=1 go run .`) then reload `/admin/courses/all`.
