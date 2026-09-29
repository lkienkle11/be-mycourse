## Purpose

Defines the format, generation, validation, and conflict-resolution rules for a course's URL slug (`courses.slug`), covering both course creation and course metadata updates, so every stored slug is a safe, unique, stable identifier usable in a public course URL.

## ADDED Requirements

### Requirement: Slug format
A course slug SHALL contain only lowercase ASCII letters `a-z`, digits `0-9`, and hyphens `-`; it SHALL NOT start or end with a hyphen; consecutive hyphens ARE allowed. The system SHALL validate every slug — whether manually supplied or generated — against the pattern `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$` before it is persisted.

#### Scenario: Valid manual slug accepted
- **WHEN** a caller supplies slug `golang-course`, `abc---123`, or a single character `a`
- **THEN** the slug passes format validation

#### Scenario: Invalid manual slug rejected
- **WHEN** a caller supplies a slug containing uppercase letters, non-ASCII characters, underscores, other special characters, or a leading/trailing hyphen (e.g. `-abc`, `abc-`, `ABC`, `abc_123`, `日本語`, `🔥course`)
- **THEN** the request is rejected with a `400` validation error and the slug is not persisted

### Requirement: Slug never null or empty in storage
The system SHALL NOT persist `NULL`, an empty string, or a whitespace-only string as a course's slug, under any code path. The database SHALL enforce this independently of application-layer validation.

#### Scenario: Database rejects an invalid slug even if application code has a bug
- **WHEN** any write attempts to set `courses.slug` to `NULL`, `''`, or a value that does not match the slug format pattern
- **THEN** the database rejects the write via a `NOT NULL` constraint and a `CHECK` constraint on format

### Requirement: Course creation accepts an optional manual slug
`POST /api/v1/courses` SHALL accept an optional `slug` field. A missing field, `null`, an empty string, or a whitespace-only string (after trimming) SHALL all be treated as "no manual slug supplied" and SHALL NOT produce a validation error.

#### Scenario: Slug omitted on create
- **WHEN** a caller creates a course without a `slug` field (or with `slug` empty/whitespace)
- **THEN** the system auto-generates a slug (see "Auto-generated slug derives from title") and creates the course with it

#### Scenario: Manual slug omitted characters trimmed
- **WHEN** a caller supplies a manual slug with leading/trailing whitespace
- **THEN** the system trims it before validation and does not otherwise alter its characters

### Requirement: Auto-generated slug derives from title
When no manual slug is supplied on create, the system SHALL derive a base slug from the course title using an ASCII transliteration of the title. If the transliterated result is empty or invalid (e.g. a title consisting only of emoji or symbols with no transliterable characters), the system SHALL fall back to a purely random slug of the form `course-{randomSuffix}`.

#### Scenario: Title transliterates successfully
- **WHEN** a course is created with title "Lập trình Golang cơ bản" and no manual slug
- **THEN** the generated base slug is an ASCII transliteration of the title (e.g. `lap-trinh-golang-co-ban`)

#### Scenario: Title has no transliterable characters
- **WHEN** a course is created with a title consisting only of emoji or symbols and no manual slug
- **THEN** the system generates a slug of the form `course-{randomSuffix}` instead of persisting an empty or non-ASCII slug

### Requirement: Manual slug conflict on create returns a recommendation
When a manually supplied slug on create already exists among active courses, the system SHALL reject the create with a structured conflict response identifying the error as a slug conflict and including a recommended alternative slug, instead of silently substituting a different slug or creating the course anyway.

#### Scenario: Manual slug already taken
- **WHEN** a caller creates a course with slug `golang-course` and an active course already has that slug
- **THEN** the system responds with an error whose machine-readable code identifies a slug conflict and whose data includes a `recommended_slug` (the requested slug plus a random suffix, confirmed available at response time) — the course is NOT created

#### Scenario: Caller retries with the recommended slug
- **WHEN** a caller resubmits create using the previously returned `recommended_slug`
- **THEN** the system treats it exactly like any other manually supplied slug: it re-validates availability at write time, and if it is still available the course is created with it

#### Scenario: The recommended slug itself is taken before the caller resubmits (rare)
- **WHEN** a caller resubmits create using a previously returned `recommended_slug`, but a concurrent request has taken that exact value in the meantime
- **THEN** the system responds with another slug-conflict error and a new recommendation, built from the same base with a fresh random suffix (e.g. `golang-course-x7k92ab` → `golang-course-x7k92ab-k39mz81`) — there is no special-cased silent auto-resolve for a resubmitted recommendation; it is handled by the ordinary manual-slug-conflict path, since the suffix keyspace makes a repeat collision astronomically unlikely in practice

### Requirement: Course update accepts an independent, omittable slug field
`PATCH /api/v1/courses/:courseId/basic-info` SHALL accept an optional `slug` field whose presence/absence is meaningful: when the field is omitted from the request, the course's slug SHALL remain unchanged, regardless of whether `title` changes in the same request. Changing `title` alone SHALL NOT change the slug.

#### Scenario: Slug field omitted on update
- **WHEN** a caller updates a course's basic info without a `slug` field, including when `title` changes
- **THEN** the course's existing slug is left unchanged

#### Scenario: Slug explicitly blanked on update is rejected
- **WHEN** a caller updates a course's basic info with `slug` present and set to `null`, an empty string, or a whitespace-only string
- **THEN** the request is rejected with a `400` validation error and no field in the request is updated... (the whole request fails, not just the slug)

### Requirement: Updating slug to its current value is a no-op, not a conflict
When the supplied update slug is identical to the course's current slug, the system SHALL treat this as no change and SHALL NOT report a conflict, even though the value already exists in the database (it belongs to this same course).

#### Scenario: Resubmitting the same slug on update
- **WHEN** a caller updates a course's basic info with `slug` equal to that course's current slug
- **THEN** the system does not error, does not alter the slug, and applies any other changed fields normally

### Requirement: Update slug conflict is auto-resolved, not reported
When a new manual slug on update collides with another active course's slug, the system SHALL automatically resolve the conflict by appending a random suffix (retrying with escalating suffix length until unique) and update the course directly — it SHALL NOT return a conflict error or a recommendation for the caller to confirm.

#### Scenario: New slug on update collides with another course
- **WHEN** a caller updates a course's slug to `advanced-golang` and another active course already has that slug
- **THEN** the system persists `advanced-golang-{randomSuffix}` (or a longer suffix if needed for uniqueness) directly, without asking the caller to confirm

### Requirement: Every slug-affecting response returns the final stored slug
Any API response for an operation that creates a course or may change its slug SHALL include the actual final slug as stored in the database, not the slug value as submitted in the request. Callers SHALL treat the response value as authoritative.

#### Scenario: Create response reflects the actual stored slug
- **WHEN** a course is created (with or without a manual slug, with or without conflict resolution)
- **THEN** the response body's slug field equals the value actually persisted to `courses.slug`

#### Scenario: Update response reflects the actual stored slug after auto-resolution
- **WHEN** a course update triggers slug auto-resolution due to a conflict
- **THEN** the response body's slug field equals the newly resolved value, not the value the caller submitted

### Requirement: Slug collision resolution uses an escalating random suffix
When the system needs to generate a collision-free variant of a base slug (auto-generation fallback, manual-conflict recommendation, or update auto-resolution), it SHALL append a suffix of lowercase ASCII letters and digits only (no hyphen, no uppercase) generated with a cryptographically secure random source. It SHALL first attempt a 7-character suffix up to 7 times; if all 7 attempts collide, it SHALL retry with an 8-character suffix (up to 7 attempts), then 9, and so on, with no fixed upper bound on suffix length.

#### Scenario: First-attempt suffix is available
- **WHEN** the system generates a 7-character random suffix for a base slug and no existing active course has that combination
- **THEN** it uses that slug immediately

#### Scenario: Repeated collisions escalate suffix length
- **WHEN** 7 consecutive 7-character suffix attempts for the same base slug all collide with existing active slugs
- **THEN** the system continues with 8-character suffix attempts (up to 7), and continues escalating length until an unused combination is found

### Requirement: A slug with an appended suffix never exceeds the storage limit
When appending a random suffix to a base slug (auto-generation fallback, manual-conflict recommendation, or update auto-resolution), the system SHALL ensure the resulting combined slug never exceeds the database column's maximum length. It SHALL truncate the base portion as needed to make room for the hyphen and suffix, rather than producing a value the database would reject. Auto-generated slugs derived from a title SHALL also be truncated to the column's maximum length before any further processing, independent of suffix concerns, since transliteration of a title can produce a result longer than the title itself.

#### Scenario: Manual slug near the maximum length conflicts and needs a suffix
- **WHEN** a caller supplies a manual slug at or near the maximum column length, and it conflicts with an existing active course's slug
- **THEN** the recommended (create) or auto-resolved (update) slug truncates the base portion as needed so the final value including the appended suffix still fits within the column's maximum length

#### Scenario: Auto-generated slug from a long or multi-byte title
- **WHEN** a course title transliterates into a slug longer than the column's maximum length (e.g. a title using a script where each character expands to several ASCII characters)
- **THEN** the generated base slug is truncated to the maximum length before format validation and any collision resolution, so it is never rejected by the database for being too long

### Requirement: Slug uniqueness is enforced at the database, not just pre-checked
The system MAY pre-check slug availability with a read before writing, but the database's uniqueness constraint SHALL remain the final authority. A write that loses a race to a concurrent request with the same slug SHALL be detected via the resulting write failure (not assumed safe because an earlier read found the slug available), and the system SHALL retry with a newly generated suffix when the conflicting slug was one it was allowed to adjust (auto-generated or auto-resolved paths).

#### Scenario: Concurrent create requests race on the same manual slug
- **WHEN** two create requests submit the same manual slug at nearly the same time
- **THEN** at most one course is created with that exact slug; the other request receives a slug-conflict response (or, if resubmitted with the recommended slug, succeeds with a different final slug) — no duplicate slug value ever exists among active courses
