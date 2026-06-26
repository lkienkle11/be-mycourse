# Session summary — roster exclude platform staff (2026-06-25)

## Bug

Users with `sysadmin` and/or `admin` roles (platform staff accounts) still appeared in instructor roster list despite being platform operators, not teaching instructors.

## Fix

- `ListRoster`: instructor role **and** exclude `sysadmin` / `admin`.
- `ListRosterCandidates`: exclude users with `instructor`, `sysadmin`, or `admin`.
- `AddRosterByUserID`: reject with `ErrRosterPlatformStaffUser` when user has platform staff roles.
- Added `UserHasPlatformStaffRole` repo method; role name constants in domain.

## Quality gates

- `make check-all`: PASS

## Follow-up fix (roster-candidates 500)

After inlining role names in `rosterCandidatesBaseSQL`, SQL had no `@placeholders` but `userpicker.ListRows` still called `gorm.DB.Raw(sql, map[string]any{})`, causing PostgreSQL driver error `expected 0 arguments, got 1` when opening add-instructor modal.

- Fixed in `internal/shared/userpicker/list.go` via `gormRaw` — skip named-arg map when `len(args)==0`.
- Blast radius: `ListRosterCandidates`, `ListInstructorCandidates` (GitNexus impact: LOW, d=1).
- API smoke: `GET /instructors/roster-candidates` → code 0 after BE restart.
