# Instructor management module

_Last audited: 2026-07-02 — instructor application feature (migration `000029`): state machine A–H, `GET/PUT /me`, company snapshot, application expertise junctions, P68 `submit_blocked`, SLA 5-day `returned`._

The instructor module (`internal/instructor/`) manages the **instructor roster**, **applications** (submit / resubmit / approve / reject / return), **profiles**, **expertise** (topic/skill junctions), and **support tickets**. It uses **additive RBAC**: assigning the `instructor` role does **not** remove `learner`.

**FE contract:** `fe-mycourse/docs/instructor-admin.md`, `fe-mycourse/docs/instructor-application.md`

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
│   ├── ports.go
│   └── validate_profile.go
├── infra/
│   ├── repos.go
│   ├── repo_roster_bulk.go
│   ├── repos_load.go
│   ├── repos_map.go
│   ├── rows.go
│   └── profile_jsonb.go   # certificates, portfolio_links, rejection_history JSONB
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
```

Registered in `internal/server/router.go`: `instdelivery.RegisterRoutes(authen, h.Instructor, svc.RBAC)`.

---

## Database (migrations `000013` + compat `000015`–`000019` + feature `000029`)

| Table | Purpose |
|-------|---------|
| `users.phone` | `VARCHAR(32)` on roster list |
| `instructor_applications` | One active row per user; inline profile snapshot + review state machine columns |
| `instructor_application_topics` | Application-scoped topic snapshot (`application_id`, `topic_id`) — migration **`000029`** |
| `instructor_application_skills` | Application-scoped skill snapshot (`application_id`, `skill_id`) — migration **`000029`** |
| `instructor_profiles` | Managed profile per user (admin CRUD); same profile columns as applications after approve copy |
| `instructor_expertise_topics` | Live instructor expertise (`user_id`, `topic_id`) — promoted from application junction on approve |
| `instructor_expertise_skills` | Live instructor expertise (`user_id`, `skill_id`) |
| `instructor_tickets` | `status` open/closed |
| `instructor_ticket_messages` | Thread messages |

Physical model audit (BE-01): see **`docs/database.md` § Instructor application feature (`000029`)** for logical snapshot contract and column mapping.

Permissions **P41–P58** seeded in `000013`; **P68** `instructor_application:submit_blocked` in **`000029`**. After deploy: `go run ./cmd/syncpermissions` and `go run ./cmd/syncrolepermissions`.

---

## Instructor application — state machine (A–H)

### State resolver priority (canonical — BE + FE must match)

Approved users receive role `instructor` and therefore effective `P68` (`submit_blocked`). **State G must win over State B** when the user's application was approved through this flow.

| Order | Check | State |
|-------|-------|-------|
| 1 | Not authenticated | **A** |
| 2 | `GET /instructor-applications/me` → `review_status === "approved"` | **G** |
| 3 | `rejection_count >= 5` | **H** |
| 4 | Effective `instructor_application:submit_blocked` (P68) | **B** — pre-existing/roster instructor or system block; **not** users at step 2 |
| 5 | No application row / eligible first submit | **C** |
| 6 | `pending` within SLA | **D** |
| 7 | `returned` | **E** |
| 8 | `rejected` with `rejection_count < 5` | **F** |

**State B vs State G:**

| | State B | State G |
|---|---------|---------|
| Step | Permission gate (after API shows not approved / not H) | Application API (`review_status`) |
| Typical cause | Instructor added via roster, legacy instructor, reject-quota `user_permissions` | Just approved via `POST …/approve` on this application |
| Message | "You cannot submit at this time" | "Congratulations — application approved" |
| P68 | Yes (blocks submit) | User may still have P68 on role, but **resolver shows G first** |

Sources: `GET /instructor-applications/me`, `GET /api/v1/me/permissions`.

| State | Condition | `review_status` / gate |
|-------|-----------|------------------------|
| A | Not authenticated | — |
| G | `review_status === approved` | **checked before P68** |
| H | `rejection_count >= 5` | `rejected` |
| B | P68 and not G/H | permission |
| C | No application or eligible first submit | no row / eligible |
| D | Pending within SLA | `pending`, `now < review_due_at` |
| E | Returned (SLA exceeded) | `returned` |
| F | Rejected, `rejection_count < 5` | `rejected` |

### Review status values

| Value | Meaning |
|-------|---------|
| `pending` | Awaiting admin/sysadmin review; SLA clock active |
| `approved` | Application accepted; instructor role assigned |
| `rejected` | Admin rejected with reason; increments `rejection_count` |
| `returned` | Pending exceeded 5 days without action; data preserved; does **not** increment `rejection_count` |

### Valid transitions

```
(none) ──POST──► pending
pending ──approve──► approved
pending ──reject──► rejected
pending ──SLA job──► returned
returned ──PUT /me──► pending
rejected ──PUT /me──► pending   (only if rejection_count < 5 and not submit_blocked)
```

### SLA (5 days)

- On submit/resubmit: set `submitted_at = now`, `review_due_at = submitted_at + 5 days`, set `returned_at = NULL`.
- When `pending` and `now >= review_due_at`: transition to `returned`, set `returned_at = now`, **do not** increment `rejection_count`.
- Implementation: periodic job and/or query-time normalization in application service (BE-08).

### Submit / resubmit rules (BE-04)

| Rule | Behaviour |
|------|-----------|
| P68 `submit_blocked` | Reject `POST` and `PUT /me` |
| Already instructor role | Reject submit |
| `rejection_count >= 5` | Reject normal submit/resubmit (State H — contact admin flow) |
| `POST /instructor-applications` | **First submit only** (State C) |
| `PUT /instructor-applications/me` | **Resubmit only** from `returned` or `rejected` |
| Resubmit from `returned` | Unlimited; does not increase `rejection_count` |
| Resubmit from `rejected` | Allowed when `rejection_count < 5` |

### Approve side effects

- Idempotent `AssignRole(instructor)` (role grant includes P68 — submit blocked on future applications).
- Copy inline profile snapshot from application → `instructor_profiles` (create if missing).
- Copy `instructor_application_topics` / `instructor_application_skills` → `instructor_expertise_topics` / `instructor_expertise_skills`.
- Set `review_status = approved`.
- **FE:** after approve, `GET /me` returns `review_status=approved` → State **G**, not State B, even though user now has P68.

---

## Business rules (roster & tickets)

| Action | RBAC / behaviour |
|--------|------------------|
| Add roster (bulk) | User must exist, must not have `sysadmin` or `admin` role; assigns `instructor` role only — **learner kept** |
| List roster | Users with `instructor` role **excluding** `sysadmin` / `admin` |
| Remove roster | `RemoveRole(instructor)` only; wipe instructor-scoped rows |
| Ticket message | Rejected when ticket `closed` |
| Close ticket | `instructor_ticket:close` (P58) |

---

## API endpoints (`/api/v1`)

All routes require `Authorization: Bearer <token>` unless noted.

### Roster (`instructor_roster:*`)

| Method | Path | Permission |
|--------|------|------------|
| GET | `/instructors` | `instructor_roster:read` |
| GET | `/instructors/roster-candidates` | `instructor_roster:create` |
| POST | `/instructors/bulk` | `instructor_roster:create` |
| DELETE | `/instructors/:id` | `instructor_roster:delete` |

### Applications (`instructor_application:*`)

| Method | Path | Permission | Notes |
|--------|------|------------|-------|
| GET | `/instructor-applications/me` | `instructor_application:create` (P45) | Resolve state A–H; prefill + `rejection_history` inline |
| PUT | `/instructor-applications/me` | `instructor_application:create` (P45) | Resubmit from `returned` / `rejected` — self-service applicant endpoint; **not** `instructor_application:update` (P46) |
| GET | `/instructor-applications` | `instructor_application:read` | Query `status` (`pending`, `approved`, `rejected`, **`returned`**), `has_profile`, `page`, `per_page` |
| POST | `/instructor-applications` | `instructor_application:create` | **First submit only** → `pending` |
| GET | `/instructor-applications/:id` | `instructor_application:read` | Detail with identity + snapshot + hydrated media |
| POST | `/instructor-applications/:id/approve` | `instructor_application:approve` | |
| POST | `/instructor-applications/:id/reject` | `instructor_application:reject` | Body `{ "rejection_reason": "..." }` |
| DELETE | `/instructor-applications/:id` | `instructor_application:delete` | |

**`GET /me` response groups** (see `docs/api_swagger.yaml`):

- Identity: `user_id`, `display_name`, `email`, `avatar`
- Review: `review_status`, `can_resubmit`, `rejection_count`, `submitted_at`, `review_due_at`, `returned_at`, `rejection_reason` (latest)
- `latest_submission.profile` — full profile snapshot including company fields and `years_of_experience` enum code
- `latest_submission.topic_ids`, `latest_submission.skill_ids`
- `rejection_history[]` — `{ rejected_at, rejected_by_user_id, reviewer_display_name, reason }`
- Hydrated read models: `cv_file`, `intro_video_file` when IDs present

**List/detail admin DTO** adds `display_name`, `email`, `avatar`; company snapshot fields; topic/skill chips (joined taxonomy names); rejection history on detail.

### Profiles (`instructor_profile:*`)

Unchanged from prior contract; profile rows include company snapshot columns after **`000029`**.

### Expertise (`instructor_expertise:*`)

Under `/instructors/:id/…` — live instructor expertise (post-approve). Application expertise snapshot uses junction tables on the application row.

### Tickets

| Method | Path | Permission |
|--------|------|------------|
| GET | `/instructor-tickets` | `instructor_application:read` |
| POST | `/instructor-tickets` | `instructor_application:create` |
| POST | `/instructor-tickets/:id/close` | `instructor_ticket:close` |
| GET/POST | `/instructor-tickets/:id/messages` | read / create |

**State H contact (BE-10):** `POST /api/v1/instructor-applications/contact-admin` with body `{ "subject", "message" }` — creates an `instructor_tickets` row plus first message (P45). Response: `{ ticket_id, status }`. FE State H tab calls this endpoint after rejection quota ≥ 5.

### Permission check for submit block

`GET /api/v1/me/permissions` exposes effective permissions. FE uses P68 at resolver **step 4 only** (after G/H). Do not map P68 → State B before `review_status === "approved"`.

---

## Permissions catalog (P41–P58, P68)

| ID | Name |
|----|------|
| P41 | `instructor_roster:read` |
| P42 | `instructor_roster:create` |
| P43 | `instructor_roster:delete` |
| P44 | `instructor_application:read` |
| P45 | `instructor_application:create` |
| P46 | `instructor_application:update` — admin-side updates only; **not** used for `PUT /me` resubmit |
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
| **P68** | **`instructor_application:submit_blocked`** |

**P68 grants:** `instructor` role (migration `000029`). Users at reject quota (≥5) may receive P68 via `user_permissions` at runtime (BE-04). **admin** / **sysadmin** do not receive P68 by default.

Canonical definitions: `internal/shared/constants/permissions.go`. Grants: `internal/system/application/roles_permission.go`.

---

## Related docs

- [`docs/database.md`](../database.md) — tables, BE-01 physical model, P68, migration `000029`
- [`docs/router.md`](../router.md) — route matrix
- [`docs/logic-flow.md`](../logic-flow.md) — application state flow
- [`docs/api_swagger.yaml`](../api_swagger.yaml) — OpenAPI contract
- [`fe-mycourse/docs/instructor-admin.md`](../../../fe-mycourse/docs/instructor-admin.md) — admin UI
- [`fe-mycourse/docs/instructor-application.md`](../../../fe-mycourse/docs/instructor-application.md) — user become-instructor page
