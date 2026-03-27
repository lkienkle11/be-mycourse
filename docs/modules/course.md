# Course Module

## Business Logic

- Public course listing and detail endpoints.
- Instructors/Admin can create courses.

## Constraints

- Course owner must be instructor or admin.
- Title is required and should be normalized.

## Transaction Notes

- Create course in one transaction with slug/metadata if present.
