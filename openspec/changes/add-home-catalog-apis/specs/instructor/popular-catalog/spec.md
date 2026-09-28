## Purpose

Lets any visitor — authenticated or not — discover active instructors ranked by teaching output, powering the public Home page's "Popular Instructor" section without requiring a session.

## ADDED Requirements

### Requirement: Public popular-instructor list requires no authentication
The system SHALL serve the popular instructors list to a request that carries no Authorization header and no session cookie.

#### Scenario: Request without credentials succeeds
- **WHEN** a client calls the popular instructors endpoint with no Authorization header and no session cookie
- **THEN** the system returns HTTP 200 with the popular instructor list, not 401

### Requirement: List is ranked by published course count, tie-broken by recency
The system SHALL order instructors by their number of published courses descending, and SHALL break ties by the most recently published course.

#### Scenario: More prolific instructor ranks first
- **WHEN** instructor A has more published courses than instructor B
- **THEN** the system ranks instructor A above instructor B

#### Scenario: Tie broken by recency
- **WHEN** instructor A and instructor B have the same number of published courses
- **THEN** the system ranks whichever published a course more recently first

### Requirement: Only currently active instructors are eligible
The system SHALL exclude a user from the popular instructor list if they no longer hold the instructor role, or if their account is disabled, soft-deleted, or currently banned, even if they own published courses.

#### Scenario: Removed instructor excluded
- **WHEN** a user's instructor role has been removed
- **THEN** the system excludes them from the popular instructor list regardless of past published courses

#### Scenario: Disabled account excluded
- **WHEN** a user's account is disabled or soft-deleted
- **THEN** the system excludes them from the popular instructor list

#### Scenario: Actively banned account excluded
- **WHEN** a user's account has an active `banned_until` timestamp in the future
- **THEN** the system excludes them from the popular instructor list

### Requirement: Response is a limited, public-safe projection
The system SHALL include only display name, avatar URL, current job title (used as a subtitle), and published course count for each instructor, and SHALL NOT include email, phone, or any application/review data.

#### Scenario: No PII beyond public profile fields
- **WHEN** the popular instructor list is returned
- **THEN** each item contains only the public-safe fields and no email, phone, or application/review data

### Requirement: Result count is bounded by a limit parameter
The system SHALL accept an optional `limit` parameter, applying a default and a maximum when it is omitted or exceeds the maximum.

#### Scenario: Explicit limit respected
- **WHEN** a client requests the popular instructors list with `limit=2`
- **THEN** the system returns at most 2 items

#### Scenario: Omitted limit uses default
- **WHEN** a client requests the popular instructors list without a `limit` parameter
- **THEN** the system returns the default number of items

### Requirement: Endpoint is rate-limited and cache-backed
The system SHALL enforce the existing public rate-limit tier on this endpoint and SHALL serve repeated identical requests from a short-TTL cache without requiring the cache to be available for correctness.

#### Scenario: Excess requests are throttled
- **WHEN** a single client IP exceeds the public rate-limit tier's request budget within its window
- **THEN** the system rejects further requests with HTTP 429 until the window resets

#### Scenario: Repeated requests within TTL are served from cache
- **WHEN** two requests for the same `limit` arrive within the cache TTL
- **THEN** the system serves the second request from cache rather than re-querying the database

#### Scenario: Cache unavailable does not break the endpoint
- **WHEN** the cache backend is unreachable
- **THEN** the system still returns a correct popular instructor list by querying the database directly
