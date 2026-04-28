# Course Module


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

> **Status: Planned / Not yet implemented.**  
> This document describes the intended design based on the permission catalog and RBAC configuration. No course-related endpoints exist in the current codebase.

---

## Overview

The Course module will manage the lifecycle of courses: creation, update, deletion, and public listing. Courses are the central content entity in MyCourse.

---

## Planned Permissions

| Permission ID | Permission Name | Granted to roles |
|---|---|---|
| P5 | `course:read` | sysadmin, admin, instructor, learner |
| P6 | `course:update` | sysadmin, admin, instructor |
| P7 | `course:delete` | sysadmin, admin, instructor |
| P8 | `course:create` | sysadmin, admin |
| P9 | `course_instructor:read` | sysadmin, instructor |

---

## Planned Business Logic

- Public course listing and detail endpoints (no auth required for read).
- Instructors and Admins can create and manage courses.
- Sysadmin has full access including `course_instructor:read`.

## Planned Constraints

- Course owner must hold the `instructor` or `admin` role.
- Course title is required; slugs should be unique.
- Soft delete preferred to preserve enrollment / lesson history.

## Planned Transaction Notes

- Course creation should be atomic with metadata (slug, category).
- Batch lesson reorder/update should run in one transaction.

---

## Implementation Reference (when added)

When course APIs are implemented, update:
- `api/v1/routes.go` — add course route group under `RegisterAuthenRoutes` and/or `RegisterNotAuthenRoutes`
- `services/course.go` — business logic
- `models/course.go` — GORM model
- `dto/course.go` — request/response DTOs
- `migrations/` — new SQL migration file
- This file and `docs/architecture.md` — update the public API surface table

---

## Testing

- **Module-level / integration** tests: **`tests/`** at repo root (`tests/README.md`, root `README.md` **Testing**).
