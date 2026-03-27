# Enrollment Module

## Business Logic

- Student enrolls in a course after successful payment flow.
- Enrollment state controls lesson access.

## Constraints

- Student cannot enroll twice in the same active course.

## Transaction Notes

- Enrollment creation and payment status update should be atomic.
