# All Sprints

History and roadmap of `be-mycourse`: what is **to do**, **in progress** and **done**. It is a log, not a system description: behavior and rules live in [`FEATURES.md`](FEATURES.md), [`ARCHITECTURE.md`](ARCHITECTURE.md), [`DATA-MODEL.md`](DATA-MODEL.md) and [`SHARED.md`](SHARED.md).

Rules:

- One line per item; a `Done` row points at the OpenSpec change or the plan phase instead of restating it.
- Not read by default and exempt from the "update docs after every task" rule: open it only when a task asks about history or the roadmap.
- When work starts on a `To do` item move it to `In progress` and fill the OpenSpec change name; when it finishes move it to `Done`.
- `Owner` is `agent`, `developer` or `unassigned`. Every row below came from the product owner's own planning artifacts and is kept as written there: the former Cursor plan `.vscode/e-learning_crud_backend_8f5ca286_reset_all_pending.plan.md` (258 todos, 225 completed, 33 pending; added 2026-05-19 in `469b743`, removed in `e7a6f7b`) and the five source specs `docs-will-be-delete/*` (added `9cff099`, removed `fac0a7c`). Both stay recoverable from Git history.

## To do

Old plan Phase 02-12 (each phase had `start` / `core` / `end` todos; difficulty grows from 02 to 12). "Today" says what exists in the code now; Phases 02-05 were built later through OpenSpec changes and a different design (versioned courses), so they are listed as covered.

| Item | Owner | Notes |
|------|-------|-------|
| Phase 06 remainder: lesson subtitles (`.srt` / `.vtt`) and declared course languages | unassigned | Plan: `lesson_texts` JSONB (+ optional LZ4), `videos`, `subtitles`. Today: text and video exist (`course_sub_lesson_texts`, `course_sub_lesson_videos`); **no subtitles table or API** |
| Phase 07 remainder: video quizzes (popup questions at timestamps), quiz timers, pass scores, per-question weights | unassigned | Plan: quiz authoring with question/option and video quiz, deep nested CRUD. Today: `course_sub_lesson_quizzes` + `_quiz_options`; timers / weights / video quiz not found |
| Phase 08: course series (learning paths) | unassigned | Plan: `course_series` + ordered link tables + outcomes, `series_items` module. Today: nothing (0 commits mention series) |
| Phase 09: coupons | unassigned | Plan: `coupons` + scope junctions (course / category / series), apply-code query, rule evaluator; percentage or fixed amount, usage limits, expiry. Today: nothing |
| Phase 10: orders and order items (plus cart) | unassigned | Plan: `orders` + `order_items`, aggregate and lifecycle queries; spec `commerce/{promotions,cart,orders}`. Today: nothing |
| Phase 11 remainder: series enrollments and prerequisites | unassigned | Plan: `enrollments` + `series_enrollments` with ownership / uniqueness-safe enroll flows. Today: `course_enrollments` exists; series enrollments and prerequisites do not |
| Phase 12: quiz attempts and grading, reviews and ratings, threaded replies | unassigned | Plan: progress + attempts + reviews / replies / ratings, `sync_uid` sync. Today: `course_progress_items` keyed by `stable_content_id` exists; **attempts, ratings (course and instructor), threaded comments with user tagging do not** |
| Progress synchronization to a new course version via `sync_uid` (opt-in migrate, keep old progress if declined), resume position, real-time tracking signals | unassigned | Spec `learning_workspace/progress`. Today: progress survives a version swap by `stable_content_id`; opt-in sync flow, scroll / watch-time tracking and resume position need a check |
| Certificates (multiple types by completion criteria, dynamic HTML templates) | unassigned | Spec `learning_workspace/certificates`. Today: nothing |
| Instructor and admin analytics (revenue, active students, quiz pass rate, system totals) | unassigned | Spec Instructor / Admin sections. Today: nothing |
| SysAdmin tooling: payment keys (Stripe / VNPay / PayPal), storage and SMTP settings, instructor commission, upload limits, error log and webhook monitoring | unassigned | Spec SysAdmin section. Today: `system_app_config` and Bunny webhook handling exist; the admin UI / API for these settings and log monitoring do not |
| Course metadata beyond level / outcomes / skills / tags: tools used, prerequisites, co-instructor details | unassigned | Plan Phase 04 / 11 and instructor spec. Collaborators exist; the rest was not found |
| Functional backend skill for the SKILLS rule (DDD folder structure, transactions, error handling): none exists in project or agent folders | developer | Found by the inventory of `shrink-agents-md-under-32kib`; project skills are `postgresql-optimization`, `cache-strategy-implementer`, `logic-n-1-optimize` (non-functional) plus three workflow skills |

## In progress

| Item | Owner | OpenSpec change |
|------|-------|-----------------|
| Shrink `AGENTS.md` under 32 KiB, add `session-context` / `branch-rename` / `docs-sync` skills, fix Commit rule | agent | `shrink-agents-md-under-32kib` |
| Unify taxonomy slug management (96 / 99 tasks) | unassigned | `unify-taxonomy-slug-management` |
| `/me` roles field (40 / 41 tasks) | unassigned | `add-me-roles-field` |

## Done

Completed and committed, not yet archived (OpenSpec status `complete`): `add-resource-scoped-role-gate` (21 / 21, created 2026-09-09), `revert-course-authorization-extension` (30 / 30, 2026-09-09), `disable-dev-deploy-job` (13 / 13, 2026-09-11), `replace-course-collaborator-with-role-gate` (35 / 35), `add-home-catalog-apis` (50 / 50, 2026-09-27), `rework-course-slug-management` (66 / 66, 2026-09-28), `add-user-slug-management` (65 / 65, 2026-10-15), `restructure-docs-and-agents` (53 / 53, 2026-10-21; docs collapsed to four files).

| Date | Item | Summary | Source |
|------|------|---------|--------|
| 2026-04-23 | Phase 01 taxonomy | Taxonomy migration, models, DTOs, repositories, services and routes; permissions P14-P25 | old plan Phase 01 |
| 2026-04-25 | Phase Sub 01 | `core/entities` moved to `pkg/entities`; `pkg/logic` kept; package refactor | `IMPLEMENTATION_PLAN_EXECUTION.md` (history) |
| 2026-04-26 | Phase Sub 02 | Media upload domain, cloud-only media (no DB persistence), helpers split (helper vs util), media clients read settings, no `getenv` outside allowed path | same |
| 2026-04-26 | Phase Sub 03 | Upload cap 2 GiB per file | same |
| 2026-04-27 | Phase Sub 04 | B2/CDN URL, object keys, Bunny split, provider errcodes, status / webhook / metadata / env / audit | same |
| 2026-04-27 | Phase Sub 05 | Media follow-up (tasks 01-15) | same |
| 2026-04-28 | Phase Sub 06 | Orphan safety, reuse, deferred cleanup | same |
| 2026-04-29 | Phase Sub 07 | Orphan image cleanup | same |
| 2026-04-30 | Phase Sub 08 | Server-owned metadata / kind reset | same |
| 2026-05-01 | Phase Sub 09-10 | Bunny `UploadFileResponse` parity; cloud SDK init from `MediaSetting` | same |
| 2026-05-02 | Phase Sub 11-12 | WebP CGO + gate, executable denylist; public media without `origin_url` | same |
| 2026-05-03 | Phase Sub 13 | DB table names in `constants/dbschema_name.go` + `dbschema` | same |
| 2026-05-04 | Phase Sub 14 | Taxonomy image and user avatar via `media_files` FK | same |
| 2026-05-08 | Phase Sub 15-16 | WebP `ImageSizeFromPayload` fix; Bunny webhook reset and hardening | same |
| 2026-05-09 | Phase Sub 17 | Batch multipart media upload | same |
| 2026-05-11 | Phase Sub 18-19 | Registration confirmation email limits; Uber Zap structured logging with ELK-ready sinks | same |
| 2026-04 to 2026-10 | Course domain (old Phase 02-05 scope) | Courses, versions, sections, lessons, sub-lessons, collaborators, edit leases, enrollments and progress tables, built through later OpenSpec changes | `docs/FEATURES.md` `course` |
