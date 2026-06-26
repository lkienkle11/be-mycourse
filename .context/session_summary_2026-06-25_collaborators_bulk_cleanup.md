# Session Summary — Collaborators single-add cleanup (BE)

**Date:** 2026-06-25

## Verified removed (no code remnants)

- `POST /api/v1/courses/:courseId/collaborators` route/handler/DTO/service `AddCollaborator`
- Only bulk add remains: `POST …/collaborators/bulk`
- ~~Internal `AddCollaboratorByUserID` kept~~ — **superseded** by batch `AddCollaboratorsBulk` in `repo_collaborators_bulk.go` (see `.context/session_summary_2026-06-25_collaborators_bulk_review_v2.md`)

## Docs synced

- `requirements.md`, `return_types.md`, `curl_api.md` — bulk-only add documented

## Quality gates

- `make test-all` — PASS
- `make check-all` — PASS
