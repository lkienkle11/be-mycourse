# Enrollment Module

> **Status: Planned / Not yet implemented.**  
> This document describes the intended design. No enrollment-related endpoints exist in the current codebase.

---

## Overview

The Enrollment module manages the relationship between a learner and a course. An enrollment record is created after a successful payment or free-course acquisition, and its state controls lesson access.

---

## Planned Business Logic

- A learner enrolls in a course after a successful payment flow.
- Enrollment status controls access to locked lessons.
- Duplicate enrollment (same learner + same active course) is prevented.

## Planned Constraints

- A learner cannot enroll twice in the same active course.
- Enrollment must be linked to either a completed payment or a free-course grant.

## Planned Transaction Notes

- Enrollment creation and payment status update must be atomic to prevent partial state.

---

## Implementation Reference (when added)

When enrollment APIs are implemented, update:
- `api/v1/routes.go` — add enrollment route group
- `services/enrollment.go` — business logic
- `models/enrollment.go` — GORM model
- `dto/enrollment.go` — request/response DTOs
- `migrations/` — new SQL migration
- This file and `docs/architecture.md`
