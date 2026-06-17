# Phase 1 — repo_admin ListAdminCourses 500 (2026-06-17)

## Symptom
`GET /api/v1/course-admin/courses` → 500  
`repo_admin.go:52` — `expected 0 arguments, got 1`

## Root cause
`ListAdminCourses` always calls `Raw(q, args)` with `args := map[string]any{}`.
When `ApprovedOnly=false`, SQL has no `@approved_status` placeholder.
GORM still counts the empty map as 1 bind argument → mismatch.

## Fix
Call `Raw(q)` when no named params; `Raw(q, args)` only when `ApprovedOnly`.

## Risk: LOW — infra query only
