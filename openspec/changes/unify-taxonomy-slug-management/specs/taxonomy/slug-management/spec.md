## Purpose

Defines the format, generation, validation, and conflict-resolution rules for the `slug` field on taxonomy root entities — course levels, course topics, course skills, and tags — covering both creation and metadata updates, so every stored slug is a safe, unique, stable identifier, consistent with the standard already established for course slugs.

## ADDED Requirements

### Requirement: Slug format
A taxonomy root entity's slug SHALL contain only lowercase ASCII letters `a-z`, digits `0-9`, and hyphens `-`; it SHALL NOT start or end with a hyphen; consecutive hyphens ARE allowed. The system SHALL validate every slug — whether manually supplied or generated — against the pattern `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$` before it is persisted.

#### Scenario: Valid manual slug accepted
- **WHEN** a caller supplies slug `beginner-level`, `abc---123`, or a single character `a` when creating or updating a course level, course topic, course skill, or tag
- **THEN** the slug passes format validation

#### Scenario: Invalid manual slug rejected
- **WHEN** a caller supplies a slug containing uppercase letters, non-ASCII characters, underscores, other special characters, or a leading/trailing hyphen (e.g. `-abc`, `abc-`, `ABC`, `abc_123`, `日本語`, `🔥topic`)
- **THEN** the request is rejected with a `400` validation error and the slug is not persisted

### Requirement: Slug never null or empty in storage
The system SHALL NOT persist `NULL`, an empty string, or a whitespace-only string as a course level's, course topic's, course skill's, or tag's slug, under any code path. The database SHALL enforce this independently of application-layer validation.

#### Scenario: Database rejects an invalid slug even if application code has a bug
- **WHEN** any write attempts to set a course level's, course topic's, course skill's, or tag's `slug` column to `NULL`, `''`, or a value that does not match the slug format pattern
- **THEN** the database rejects the write via a `NOT NULL` constraint and a `CHECK` constraint on format

### Requirement: Creation accepts an optional manual slug
The create endpoint for a course level, course topic, course skill, or tag SHALL accept an optional `slug` field. A missing field, `null`, an empty string, or a whitespace-only string (after trimming) SHALL all be treated as "no manual slug supplied" and SHALL NOT produce a validation error.

#### Scenario: Slug omitted on create
- **WHEN** a caller creates a course level, course topic, course skill, or tag without a `slug` field (or with `slug` empty/whitespace)
- **THEN** the system auto-generates a slug from `name` (see "Auto-generated slug derives from name") and creates the entity with it

#### Scenario: Manual slug whitespace trimmed
- **WHEN** a caller supplies a manual slug with leading/trailing whitespace
- **THEN** the system trims it before validation and does not otherwise alter its characters

### Requirement: Auto-generated slug derives from name
When no manual slug is supplied on create, the system SHALL derive a base slug from the entity's `name` using the existing name-to-slug transliteration already used by this module. If the transliterated result is empty or does not satisfy the slug format (for example, a name using a script this transliteration cannot represent in ASCII), the system SHALL fall back to a purely random slug prefixed by the entity kind (`level-{randomSuffix}`, `topic-{randomSuffix}`, `skill-{randomSuffix}`, or `tag-{randomSuffix}`).

#### Scenario: Name transliterates successfully
- **WHEN** a course topic is created with name "Lập trình Golang" and no manual slug
- **THEN** the generated base slug is an ASCII transliteration of the name

#### Scenario: Name has no transliterable characters
- **WHEN** a tag is created with a name consisting only of a script the transliteration cannot represent in ASCII, and no manual slug
- **THEN** the system generates a slug of the form `tag-{randomSuffix}` instead of persisting an empty or non-ASCII slug

### Requirement: Manual slug conflict on create returns a recommendation
When a manually supplied slug on create already exists among active rows of the same entity type, the system SHALL reject the create with a structured conflict response identifying the error as a slug conflict and including a recommended alternative slug, instead of silently substituting a different slug or creating the entity anyway.

#### Scenario: Manual slug already taken
- **WHEN** a caller creates a course topic with slug `web-development` and an active course topic already has that slug
- **THEN** the system responds with an error whose machine-readable code identifies a slug conflict and whose data includes a `recommended_slug` (the requested slug plus a random suffix, confirmed available at response time) — the course topic is NOT created

#### Scenario: Caller retries with the recommended slug
- **WHEN** a caller resubmits create using the previously returned `recommended_slug`
- **THEN** the system treats it exactly like any other manually supplied slug: it re-validates availability at write time, and if it is still available the entity is created with it

### Requirement: Update accepts an independent, omittable slug field
The update endpoint for a course level, course topic, course skill, or tag SHALL accept an optional `slug` field whose presence/absence is meaningful: when the field is omitted from the request, the entity's slug SHALL remain unchanged, regardless of whether `name` changes in the same request. Changing `name` alone SHALL NOT change the slug.

#### Scenario: Slug field omitted on update
- **WHEN** a caller updates a course level's, course topic's, course skill's, or tag's `name` without including a `slug` field
- **THEN** the entity's existing slug is left unchanged

#### Scenario: Slug explicitly blanked on update is rejected
- **WHEN** a caller updates a course level, course topic, course skill, or tag with `slug` present and set to `null`, an empty string, or a whitespace-only string
- **THEN** the request is rejected with a `400` validation error and no field in the request is updated

### Requirement: Updating slug to its current value is a no-op, not a conflict
When the supplied update slug is identical to the entity's current slug, the system SHALL treat this as no change and SHALL NOT report a conflict, even though the value already exists in the database (it belongs to this same row).

#### Scenario: Resubmitting the same slug on update
- **WHEN** a caller updates a course topic with `slug` equal to that course topic's current slug
- **THEN** the system does not error, does not alter the slug, and applies any other changed fields normally

### Requirement: Update slug conflict is auto-resolved, not reported
When a new manual slug on update collides with another active row's slug (of the same entity type), the system SHALL automatically resolve the conflict by appending a random suffix (retrying with escalating suffix length until unique) and update the entity directly — it SHALL NOT return a conflict error or a recommendation for the caller to confirm.

#### Scenario: New slug on update collides with another row
- **WHEN** a caller updates a tag's slug to `beginner` and another active tag already has that slug
- **THEN** the system persists `beginner-{randomSuffix}` (or a longer suffix if needed for uniqueness) directly, without asking the caller to confirm

### Requirement: Every slug-affecting response returns the final stored slug
Any API response for an operation that creates a course level, course topic, course skill, or tag, or may change its slug, SHALL include the actual final slug as stored in the database, not the slug value as submitted in the request.

#### Scenario: Update response reflects the actual stored slug after auto-resolution
- **WHEN** an update triggers slug auto-resolution due to a conflict
- **THEN** the response body's slug field equals the newly resolved value, not the value the caller submitted

### Requirement: Slug collision resolution uses an escalating random suffix
When the system needs to generate a collision-free variant of a base slug (auto-generation fallback, manual-conflict recommendation, or update auto-resolution), it SHALL append a suffix of lowercase ASCII letters and digits only, generated with a cryptographically secure random source, first attempting a 7-character suffix up to 7 times, then escalating to longer suffixes with no fixed upper bound — reusing the same collision-resolution mechanism already established for course slugs.

#### Scenario: Repeated collisions escalate suffix length
- **WHEN** 7 consecutive 7-character suffix attempts for the same base slug all collide with existing active rows of the same entity type
- **THEN** the system continues with 8-character suffix attempts, and continues escalating length until an unused combination is found

### Requirement: A slug with an appended suffix never exceeds the storage limit
When appending a random suffix to a base slug, the system SHALL ensure the resulting combined slug never exceeds the database column's maximum length, truncating the base portion as needed. Auto-generated slugs derived from a name SHALL also be truncated to the column's maximum length before any further processing.

#### Scenario: Manual slug near the maximum length conflicts and needs a suffix
- **WHEN** a caller supplies a manual slug at or near the maximum column length that conflicts with an existing active row
- **THEN** the recommended (create) or auto-resolved (update) slug truncates the base portion as needed so the final value including the appended suffix still fits within the column's maximum length

### Requirement: Slug uniqueness is enforced at the database, not just pre-checked
The system MAY pre-check slug availability with a read before writing, but the database's uniqueness constraint SHALL remain the final authority. A write that loses a race to a concurrent request with the same slug SHALL be detected via the resulting write failure, and the system SHALL retry with a newly generated suffix when the conflicting slug was one it was allowed to adjust (auto-generated or auto-resolved paths).

#### Scenario: Concurrent create requests race on the same manual slug
- **WHEN** two create requests submit the same manual slug for the same entity type at nearly the same time
- **THEN** at most one entity is created with that exact slug; the other request receives a slug-conflict response — no duplicate slug value ever exists among active rows of that entity type

#### Scenario: Duplicate-key write failure is mapped to a clean error, not leaked
- **WHEN** a write attempt fails because of a unique-constraint violation on a taxonomy slug column
- **THEN** the system recognizes this failure and either retries with a new suffix (auto path) or maps it to the structured slug-conflict response (manual create path) — it SHALL NOT let the raw database error message reach the client
