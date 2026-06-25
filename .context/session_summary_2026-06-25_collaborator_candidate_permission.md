# Session Summary — P67 Collaborator Candidate Permission

**Date:** 2026-06-25

## Problem

`GET /courses/:courseId/instructor-candidates` reused `course_instructor:read` (P9). Picker replaced global `instructor_roster:read` (P41) and needs its own scoped permission.

## Implemented

- **P67** `course_collaborator_candidate:read`
- Migration `000027_course_collaborator_candidate_permission` — grants to **sysadmin**, **admin**, and **instructor** (`000028` backfills admin if needed)
- Route `instructor-candidates` uses `CourseCollaboratorCandidateRead`
- `roles_permission.go` synced (**sysadmin**, **admin**, **instructor**)
- FE: `PERMISSIONS` / `PERMISSION_IDS` + `PermissionGate` on add/picker in `CourseCollaboratorsTab`

## Quality gates

- `make test-all`, `make check-all` — PASS
- `ruby scripts/generate-apidog-postman.rb` — PASS

## Docs synced (no credentials in committed files)

- BE: `docs/router.md`, `docs/modules/course.md`, `docs/database.md`, `docs/api_swagger.yaml`, `docs/curl_api.md`, `docs/api-overview.md`, `docs/requirements.md`, `docs/reusable-assets.md`, `docs/course-collaboration-handoff-2026-06-04.md`, `migrations/README.md`
- FE: `docs/api-using.md`, `docs/api-overview.md`, `docs/reusable-assets.md`, `docs/course-collaboration-handoff-2026-06-04.md`, `docs/components.md`, `docs/folder-structure.md`, `docs/modules.md`
