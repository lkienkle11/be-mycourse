# Session Summary — Collaborators bulk review fixes v2 (BE)

**Date:** 2026-06-25  
**Source:** `temporary-docs/review-code-chua-commit/bao-cao-review.md`

## Review findings addressed

1. **Submit validation N+1 (Medium)** — `validateDraftCollaborators` batch-loads user snapshots via `loadCollaboratorAccessSnapshots` + eligibility via `instructorUserIDSet`; in-memory loop only.
2. **Bulk path tests (Low)** — `planBulkCollaboratorWrites` unit tests: all success, partial success, all failed, failure message. Service tests: input normalization + empty input.
3. **Removed dead wrapper** — `userIsInstructor` deleted (unused after batch refactor); all callers use `instructorUserIDSet` directly.

## Current implementation (final state)

1. **Bulk collaborator repo** — `repo_collaborators_bulk.go`: single transaction; `planBulkCollaboratorWrites` classifies insert/update/failed; batch `UPDATE id IN (…)` + `CreateInBatches` insert; `loadCollaboratorsByUserIDs` hydrate once.
2. **Eligibility rule** — `sqlCollaboratorEligibleRoleIN` + `instructorUserIDSet` shared by bulk add and submit validation.
3. **Submit validation** — `validateDraftCollaborators` + `loadCollaboratorAccessSnapshots` (batch, no N+1).
4. **API** — bulk-only add (`POST …/collaborators/bulk`); legacy single POST removed.

## Quality gates

- `make check-all` — PASS (2026-06-25)
- `npx gitnexus analyze --embeddings` — sync after code changes

## Docs synced

- `docs/modules/course.md`, `docs/reusable-assets.md`, `docs/course-collaboration-handoff-2026-06-04.md`
