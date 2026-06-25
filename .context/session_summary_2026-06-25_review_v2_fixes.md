# Session summary — review v2 fixes (2026-06-25)

## Review items fixed (priority 3.1 → 3.4)

### 3.1 Bulk API error safety (BE)
- `AddRosterBulk` returns `(RosterBulkResult, error)`; only whitelisted business errors go to `failed[]`.
- Infrastructure errors abort with HTTP 500 via `mapInstructorError`.
- `rosterBulkClientMessage` + unit test in `service_roster_bulk_test.go`.

### 3.2 Partial-success candidate refresh (FE)
- `UserMultiSelectPickerDialog.onAfterPartialSuccess` revalidates SWR candidate list after partial bulk add.
- `UserMultiSelectPickerFeatureDialog` calls `candidates.mutate()` for roster picker.

### 3.3 Picker wrapper dedup (FE)
- New `UserMultiSelectPickerFeatureDialog` — shared wiring for roster + collaborator pickers.
- Feature wrappers reduced to ~40 lines each; jscpd clone cleared.

### 3.4 Constant dependency direction (FE)
- `USER_PICKER_PER_PAGE` moved to `src/constants/user-picker.ts`.
- Hook imports from constants, not UI component.

### Prior bug (roster-candidates 500)
- `userpicker.gormRaw` skips empty named-arg map when SQL has no `@placeholders`.

## Quality gates
- BE `make check-all`: PASS
- FE `npm run check-all`: PASS

## Docs
- `docs/modules/instructor.md`, `docs/reusable-assets.md` (both repos)
