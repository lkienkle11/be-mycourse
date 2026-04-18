# Lesson Module

> **Status: Planned / Not yet implemented.**  
> This document describes the intended design. No lesson-related endpoints exist in the current codebase.

---

## Overview

The Lesson module will manage the ordered content items within a course. Lesson visibility is gated on enrollment status and per-lesson preview settings.

---

## Planned Business Logic

- List lessons within a course, ordered by sequence index.
- Authenticated learners with an active enrollment can access all lessons.
- Preview lessons are accessible without enrollment.
- Instructors / Admins can create, update, reorder, and delete lessons.

## Planned Constraints

- A lesson belongs to exactly one course.
- Learners can access locked lessons only after active enrollment.
- Lesson `order_index` must be unique within a course.

## Planned Transaction Notes

- Batch lesson reorder/update should run in a single transaction.
- Lesson deletion should cascade or nullify references in enrollment progress records.

---

## Implementation Reference (when added)

When lesson APIs are implemented, update:
- `api/v1/routes.go` — add lesson route group
- `services/lesson.go` — business logic
- `models/lesson.go` — GORM model
- `dto/lesson.go` — request/response DTOs
- `migrations/` — new SQL migration
- This file and `docs/architecture.md`
