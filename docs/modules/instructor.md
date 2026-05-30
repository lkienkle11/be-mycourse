# Instructor management module

_Last audited: 2026-05-29 (`internal/instructor/`, migration `000013_instructor_management`)._

The instructor module (`internal/instructor/`) manages the **instructor roster**, **applications** (submit / approve / reject), **profiles**, **expertise** (topic/skill junctions), and **support tickets**. It uses **additive RBAC**: assigning the `instructor` role does **not** remove `learner`.

**FE contract:** `fe-mycourse/docs/instructor-admin.md`

---

## Directory layout

```
internal/instructor/
├── domain/
│   ├── instructor.go      # Entities, review/ticket statuses, profile payload, certificates
│   ├── errors.go          # ErrRejectionReasonRequired, ErrApplicationNotPending, ErrTicketClosed
│   └── repository.go      # Repository interface (roster, apps, profiles, expertise, tickets)
├── application/
│   ├── service.go         # InstructorService facade
│   ├── service_roster.go
│   ├── service_applications.go
│   ├── service_profiles.go
│   ├── service_expertise.go
│   ├── service_tickets.go
│   ├── ports.go           # User lookup, RBAC role manager, media validator, avatar hydrate
│   └── validate_profile.go
├── infra/
│   ├── repos.go           # GormRepository
│   ├── repos_load.go
│   ├── repos_map.go
│   ├── rows.go
│   └── profile_jsonb.go   # certificates JSONB (not in domain)
└── delivery/
    ├── handler.go
    ├── handler_helpers.go
    ├── handler_mutations.go
    ├── handler_profile.go
    ├── handler_expertise_topic.go
    ├── handler_expertise_skill.go
    ├── handler_ticket.go
    ├── routes.go
    └── dto.go

internal/server/
├── wire_instructor.go
├── wire_instructor_adapters.go
└── wire_core.go             # shared core wiring (RBAC, auth, media, taxonomy, …)

internal/shared/mediaquery/hydrate.go   # avatar URL hydration (no media/domain import)
```

Registered in `internal/server/router.go`: `instdelivery.RegisterRoutes(authen, h.Instructor, svc.RBAC)`.

---

## Database (migration `000013`)

| Table | Purpose |
|-------|---------|
| `users.phone` | `VARCHAR(32)` on roster list |
| `instructor_applications` | One active row per user; `review_status` pending/approved/rejected; profile columns inline |
| `instructor_profiles` | Managed profile per user (admin CRUD) |
| `instructor_expertise_topics` | `(user_id, topic_id)` → `course_topics` |
| `instructor_expertise_skills` | `(user_id, skill_id)` → `course_skills` |
| `instructor_tickets` | `status` open/closed |
| `instructor_ticket_messages` | Thread messages; blocked when ticket closed |

Permissions **P41–P58** seeded in the same migration. Role grants:

- **sysadmin**, **admin**: P41–P58
- **instructor**: P45, P47, P49, P55, P56, P57, P58 (plus pre-existing catalog from `roles_permission.go`)
- **learner**: P45 (submit application / create ticket)

After deploy: `go run ./cmd/syncpermissions` and `go run ./cmd/syncrolepermissions` so `roles_permission.go` stays aligned.

---

## Business rules

| Action | RBAC / behaviour |
|--------|------------------|
| Add roster by email | User must exist; `AssignRole(instructor)` only — **learner kept** |
| Remove roster | `RemoveRole(instructor)` only; wipe instructor-scoped rows; user account kept |
| Submit application | `review_status = pending`; **no role change** |
| Approve | Idempotent `AssignRole(instructor)`; `approved` |
| Reject | `rejection_reason` required (1–2000 chars); **no** instructor role |
| Ticket message | Rejected when ticket `closed` |
| Close ticket | `instructor_ticket:close` (instructor); admins use read/create per ticket routes |

---

## API endpoints (`/api/v1`)

All routes require `Authorization: Bearer <token>` unless noted.

### Roster (`instructor_roster:*`)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/instructors` | `instructor_roster:read` |
| POST | `/instructors` | `instructor_roster:create` — body `{ "email": "..." }` |
| DELETE | `/instructors/:id` | `instructor_roster:delete` — `:id` = user id |

List returns `id`, `full_name`, `email`, `phone`, `avatar` (hydrated URL).

### Applications (`instructor_application:*`)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/instructor-applications` | `instructor_application:read` — query `status`, `has_profile`, pagination |
| POST | `/instructor-applications` | `instructor_application:create` — submit → pending |
| GET | `/instructor-applications/:id` | `instructor_application:read` |
| POST | `/instructor-applications/:id/approve` | `instructor_application:approve` |
| POST | `/instructor-applications/:id/reject` | `instructor_application:reject` — body `{ "rejection_reason": "..." }` |
| DELETE | `/instructor-applications/:id` | `instructor_application:delete` |

Application responses now include user identity fields for admin UI popups:
`full_name` and `avatar` (hydrated URL, empty string when unavailable).

### Profiles (`instructor_profile:*`)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/instructor-profiles` | `instructor_profile:read` |
| GET | `/instructor-profiles/me` | `instructor_profile:read` |
| GET | `/instructor-profiles/:id` | `instructor_profile:read` — user id |
| POST | `/instructor-profiles` | `instructor_profile:create` — upsert |
| PATCH | `/instructor-profiles/:id` | `instructor_profile:update` |
| DELETE | `/instructor-profiles/:id` | `instructor_profile:delete` |

Profile responses include `full_name` and `avatar` alongside `profile` payload data.

CV / intro video file IDs validated via media service (PDF/images/video policy in application layer).

### Expertise (`instructor_expertise:*`)

Under `/instructors/:id/…` (`:id` = user id):

| Method | Path | Permission |
|--------|------|------------|
| GET/POST | `…/expertise/topics` | read / create |
| DELETE | `…/expertise/topics/:topicRowId` | delete — junction row id |
| GET/POST | `…/expertise/skills` | read / create |
| DELETE | `…/expertise/skills/:skillRowId` | delete |

POST body: `{ "topic_id": N }` or `{ "skill_id": N }`.

### Tickets

Uses `instructor_application:read` / `:create` for list, create, messages; close uses `instructor_ticket:close`.

| Method | Path | Permission |
|--------|------|------------|
| GET | `/instructor-tickets` | `instructor_application:read` |
| POST | `/instructor-tickets` | `instructor_application:create` |
| POST | `/instructor-tickets/:id/close` | `instructor_ticket:close` |
| GET | `/instructor-tickets/:id/messages` | `instructor_application:read` |
| POST | `/instructor-tickets/:id/messages` | `instructor_application:create` |

### Stubs (coming soon)

| GET | `/instructor-stubs/assignments` | `instructor_profile:read` |
| GET | `/instructor-stubs/activity-log` | `instructor_profile:read` |

Returns standard “coming soon” envelope until implemented.

---

## Permissions catalog (P41–P58)

| ID | Name |
|----|------|
| P41 | `instructor_roster:read` |
| P42 | `instructor_roster:create` |
| P43 | `instructor_roster:delete` |
| P44 | `instructor_application:read` |
| P45 | `instructor_application:create` |
| P46 | `instructor_application:update` |
| P47 | `instructor_application:delete` |
| P48 | `instructor_application:approve` |
| P49 | `instructor_application:reject` |
| P50 | `instructor_profile:read` |
| P51 | `instructor_profile:create` |
| P52 | `instructor_profile:update` |
| P53 | `instructor_profile:delete` |
| P54 | `instructor_expertise:read` |
| P55 | `instructor_expertise:create` |
| P56 | `instructor_expertise:update` |
| P57 | `instructor_expertise:delete` |
| P58 | `instructor_ticket:close` |

Canonical definitions: `internal/shared/constants/permissions.go`. Grants: `internal/system/application/roles_permission.go`.

---

## Related docs

- [`docs/database.md`](../database.md) — tables, P41–P58 matrix, migration `000013`
- [`docs/curl_api.md`](../curl_api.md) — §13 Instructor (smoke curls)
- [`docs/folder-structure.md`](../folder-structure.md) — `internal/instructor/`
- [`fe-mycourse/docs/instructor-admin.md`](../../../fe-mycourse/docs/instructor-admin.md) — admin UI
