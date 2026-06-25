# Session summary — remove POST /instructors (single add) (2026-06-25)

## Why

Roster add UI uses multi-select picker + `POST /instructors/bulk` only. Keeping `POST /instructors` (email or single `user_id`) duplicated the API surface with no remaining callers.

## Backend

- Removed route `POST /api/v1/instructors` and handler `addRoster`.
- Removed DTO `addRosterRequest` and service method `AddRosterByEmail`.
- Kept `AddRosterByUserID` (used internally by `AddRosterBulk`).
- Docs/swagger/router/modules/curl/api-dog updated.

## Frontend

- Removed `addInstructorRosterService` and `AddRosterPayload`.
- Roster page already uses `addInstructorRosterBulkService` only.

## Quality gates

- `make check-all` (BE): PASS
- `npm run check-all` (FE): PASS
- GitNexus re-index: PASS
