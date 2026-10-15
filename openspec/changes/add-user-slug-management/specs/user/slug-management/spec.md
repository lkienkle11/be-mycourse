## Purpose

Defines the format, generation, self-service change, conflict-resolution, reservation and exposure rules for a user's URL slug (`users.slug`), so every account has a safe, unique, stable identifier that later features (instructor pages, course cards, share links) can use instead of the internal UUID.

## ADDED Requirements

### Requirement: Slug format
A user slug SHALL contain only lowercase ASCII letters `a-z`, digits `0-9` and hyphens `-`, SHALL NOT start or end with a hyphen, and SHALL be at most 255 characters. Every slug, whether generated or supplied, SHALL be validated against `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$` before it is persisted.

#### Scenario: Valid manual slug accepted
- **WHEN** a user submits slug `nguyen-van-a`, `a1`, or `abc---123`
- **THEN** the slug passes format validation

#### Scenario: Invalid manual slug rejected
- **WHEN** a user submits a slug with uppercase letters, non-ASCII characters, underscores, spaces, a leading or trailing hyphen, or more than 255 characters
- **THEN** the request is rejected with a `400` validation error and the slug is not persisted

### Requirement: Slug generated automatically at user creation
Every user created by email registration or by OAuth sign-up SHALL receive a slug at creation time, derived from the user's display name by ASCII transliteration. If the transliterated result is empty or invalid, the base SHALL be `user`. If the base is already taken by an active user, is a reserved word, or the fallback `user` is used, the system SHALL append `-` plus a random suffix of lowercase letters and digits, starting at 7 characters and growing by one character after 7 failed attempts at each length.

#### Scenario: Display name transliterates successfully
- **WHEN** a user registers with display name "Nguyễn Văn A" and no active user holds `nguyen-van-a`
- **THEN** the user's slug is `nguyen-van-a`

#### Scenario: Display name collides with an existing slug
- **WHEN** a user registers with a display name whose base slug is already held by an active user
- **THEN** the user receives `{base}-{7 random characters}` and registration succeeds without error

#### Scenario: Display name has no transliterable characters
- **WHEN** a user registers with a display name made only of emoji or symbols, or an empty display name
- **THEN** the user's slug has the form `user-{random suffix}`

#### Scenario: Concurrent registrations race for the same slug
- **WHEN** two users with the same display name register at the same time
- **THEN** both registrations succeed and the two users receive different slugs, with no raw database error exposed to either caller

### Requirement: User can change their own slug
`PATCH /api/v1/me` SHALL accept an optional `slug` field alongside the existing fields. An omitted `slug` SHALL leave the slug unchanged. An explicit `null`, an empty string or a whitespace-only string SHALL be rejected with `400`. A value equal to the current slug SHALL be a no-op. A valid new value SHALL replace the slug, and the response and subsequent `GET /api/v1/me` calls SHALL return the new slug immediately, without waiting for the profile cache to expire.

#### Scenario: Slug omitted
- **WHEN** a user sends `PATCH /api/v1/me` without a `slug` field
- **THEN** the stored slug is unchanged

#### Scenario: Slug explicitly null or empty
- **WHEN** a user sends `"slug": null`, `"slug": ""` or `"slug": "   "`
- **THEN** the request is rejected with `400` and the stored slug is unchanged

#### Scenario: Slug set to the current value
- **WHEN** a user sends their current slug
- **THEN** the request succeeds with no change and no history entry is written

#### Scenario: Slug changed
- **WHEN** a user sends a valid, available slug
- **THEN** the slug is updated and the response contains the new slug

#### Scenario: Slug-only patch
- **WHEN** a user sends `PATCH /api/v1/me` with only `slug`
- **THEN** the slug is updated without requiring any other field

### Requirement: Slug conflict on change is auto-resolved
When the requested new slug is held by another active user, is a former slug of another user, or collides in a concurrent write, the system SHALL NOT fail the request. It SHALL append a random suffix as defined for automatic generation and store the resulting slug, and the response SHALL return the slug actually stored.

#### Scenario: Requested slug already taken
- **WHEN** a user requests a slug currently held by another active user
- **THEN** the request succeeds and the response contains `{requested}-{random suffix}`

#### Scenario: Requested slug is another user's former slug
- **WHEN** a user requests a slug that another user previously held and released by changing it
- **THEN** the request succeeds and the stored slug carries a random suffix

### Requirement: Former slugs are reserved permanently
When a user's slug changes, the previous slug SHALL be recorded and SHALL NOT be assigned to any other user afterward, either manually or automatically.

#### Scenario: Old slug cannot be claimed by someone else
- **WHEN** user A changes slug from `alice` to `alice-dev`, and later user B registers with display name "Alice" or requests slug `alice`
- **THEN** user B does not receive `alice`

#### Scenario: Former slug recorded once per change
- **WHEN** a user changes slug twice
- **THEN** both former slugs are recorded and reserved

#### Scenario: User returns to their own former slug
- **WHEN** a user changes back to a slug they previously held and nobody else holds it
- **THEN** the request succeeds with exactly that slug and the slug they leave is recorded as former

#### Scenario: Hard-deleted user releases former slugs
- **WHEN** a user account is permanently deleted
- **THEN** that user's former slugs are no longer reserved, while a soft-deleted account keeps them reserved

### Requirement: Reserved words are never assigned
The system SHALL maintain a list of reserved words that SHALL NOT be used as a user slug. A manual slug equal to a reserved word SHALL be rejected with `400`. An automatically generated base equal to a reserved word SHALL be suffixed like a collision.

#### Scenario: Manual reserved slug rejected
- **WHEN** a user requests slug `admin`, `me`, `api` or `instructors`
- **THEN** the request is rejected with `400` and the slug is not persisted

#### Scenario: Display name equals a reserved word
- **WHEN** a user registers with display name "Admin"
- **THEN** the generated slug is `admin-{random suffix}`, never `admin`

### Requirement: Slug is never null or empty after rollout
After the backfill has completed and the final constraint migration is applied, the database SHALL reject any user row whose slug is `NULL`, empty or does not match the slug format. Before that migration is applied, existing users without a slug MAY have `NULL`, and no code path SHALL write an empty string.

#### Scenario: Database rejects an invalid slug
- **WHEN** any write attempts to set `users.slug` to an empty string or a value outside the slug format
- **THEN** the database rejects the write

#### Scenario: Active slugs are unique
- **WHEN** two active (non-deleted) users would hold the same slug
- **THEN** the database rejects the second write

#### Scenario: Soft-deleted user releases the active slug
- **WHEN** a user is soft-deleted
- **THEN** the unique constraint no longer counts that row, and a former-slug reservation (if any) still applies

### Requirement: Existing users are backfilled
The system SHALL provide an operator-run batch command that assigns a slug to every existing user whose slug is `NULL`, using the same generation rules as new users. The command SHALL be safe to run repeatedly and SHALL NOT change any user that already has a slug.

#### Scenario: Backfill assigns slugs
- **WHEN** the command runs and some users have a `NULL` slug
- **THEN** each such user receives a unique, format-valid slug derived from their display name

#### Scenario: Backfill is idempotent
- **WHEN** the command runs a second time after completing
- **THEN** no user row is modified

#### Scenario: Soft-deleted users are included
- **WHEN** the command runs and soft-deleted users have a `NULL` slug
- **THEN** they also receive a slug, so the final `NOT NULL` constraint can be applied table-wide

### Requirement: Slug is exposed in user-facing responses
The user's slug SHALL be returned as `slug` in `GET /api/v1/me` and `PATCH /api/v1/me`, in popular-instructor, instructor application, instructor profile and instructor roster responses, and as `owner_slug` wherever a course response already returns the owner's display name (course lists, trending courses, continue-learning, collaborators). Existing identifier fields such as `user_id` SHALL remain unchanged.

#### Scenario: Me response includes slug
- **WHEN** an authenticated user calls `GET /api/v1/me`
- **THEN** the response contains the user's `slug` alongside the existing fields

#### Scenario: Popular instructors include slug
- **WHEN** a client calls `GET /api/v1/catalog/instructors/popular`
- **THEN** each item contains `slug` and still contains `user_id`

#### Scenario: Course owner slug included
- **WHEN** a client lists trending courses
- **THEN** each course item contains `owner_slug` alongside `owner_display_name`

#### Scenario: User without a slug yet
- **WHEN** a user row still has a `NULL` slug during rollout
- **THEN** responses return an empty string for the slug and do not fail
