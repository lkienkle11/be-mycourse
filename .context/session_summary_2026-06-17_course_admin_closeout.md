# BE Close-out — course admin permissions + trash (2026-06-17)

> **Process note:** Implementation landed before full Phase-1 discovery per `temporary-docs/tieu-chuan-check-be-fe/be-mycourse.md`. This file retroactively completes the mandatory checklist.

---

## Phase 1 — Discovery (retroactive)

### Context read
- `.context/session_summary_2026-06-17_course_admin_permissions_discovery.md`
- Prior course/trash work in branch (uncommitted)

### Docs gap (before fix)
| File | Gap |
|------|-----|
| `docs/router.md` | Routes used `admin:modify` |
| `docs/modules/course.md` | Permissions table stale |
| `docs/curl_api.md` | §14 header + §14.6 permission note stale |
| `docs/database.md` | Missing P59–P66, migration 000024 |

### Git baseline
- Branch has large uncommitted course-admin + trash scope
- `git status`: routes, permissions, roles_permission, migrations 000024, docs

### GitNexus research
| Symbol | Impact | Risk |
|--------|--------|------|
| `registerCourseAdminRoutes` | d=1: `RegisterRoutes` only | LOW |
| `AllPermissions` (P59–P66) | RBAC catalog + JWT | MEDIUM |

`gitnexus_query`: course admin routes → `RegisterRoutes`, `registerCourseAdminRoutes`, repo_admin methods.

### Reuse decisions
- Reuse `utils.RoutePermission` + `constants.AllPermissions` pattern (same as instructor P41–P58)
- Reuse migration grant pattern from `000013_instructor_management`

### Risk: **MEDIUM** (RBAC — users must re-login after migration)

---

## Phase 2 — Implementation (completed)

- `migrations/000024_course_admin_permissions.{up,down}.sql`
- `internal/shared/constants/permissions.go` — P59–P66
- `internal/system/application/roles_permission.go` — sysadmin + admin grants
- `internal/course/delivery/routes.go` — granular guards

---

## Phase 3 — Quality + close-out

### Quality gates
| Command | Result |
|---------|--------|
| `make test-all` | **PASS** |
| `make check-all` | **PASS** (fmt, test-all, build-nocgo, build CGO) |

### GitNexus
- `gitnexus_detect_changes({ scope: "all" })` — run; large branch diff (CRITICAL risk = whole branch, not this task alone)

### Docs audit
```bash
rg -i 'admin:modify.*review|versionId' docs/   # clean after fix
```
Fixed: `docs/modules/course.md`, `docs/curl_api.md` §14 header

### Postman
- API paths unchanged (permission-only) — **no regenerate required**

### Manual verification
- Server started locally with `MIGRATE=1` — migration 000024 applied, routes registered
- Curl smoke (login + course-admin) — **skipped by operator** (network approval declined)

### Deploy notes
- Run `MIGRATE=1` on deploy
- Users with old JWT must **re-login** for P59–P66

---

## Checklist cuối giai đoạn

- [x] Task scope (permissions + routes + docs)
- [x] `make test-all` PASS
- [x] `make check-all` PASS
- [x] GitNexus detect_changes run
- [x] Docs audit clean (stale `admin:modify` review refs removed)
- [ ] Manual curl smoke — operator skipped
- [x] Session summary (this file)
