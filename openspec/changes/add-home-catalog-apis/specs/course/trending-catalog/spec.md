## Purpose

Lets any visitor — authenticated or not — discover currently available courses ranked by recency, powering the public Home page's "Trending Course" section without requiring a session and without leaking draft, unpublished, or workflow-internal course data.

## ADDED Requirements

### Requirement: Public trending list requires no authentication
The system SHALL serve the trending courses list to a request that carries no Authorization header and no session cookie.

#### Scenario: Request without credentials succeeds
- **WHEN** a client calls the trending courses endpoint with no Authorization header and no session cookie
- **THEN** the system returns HTTP 200 with the trending course list, not 401

### Requirement: List reflects only published, non-trashed courses
The system SHALL exclude any course that has no published version, is trashed, or is soft-deleted from the trending list.

#### Scenario: Draft-only course excluded
- **WHEN** a course has never had a version approved (no published version)
- **THEN** the system excludes it from the trending list

#### Scenario: Trashed course excluded
- **WHEN** a previously published course has been moved to trash
- **THEN** the system excludes it from the trending list

### Requirement: List is ordered by course creation recency
The system SHALL order the trending list by course creation time, most recent first.

#### Scenario: Newest course ranks first
- **WHEN** two published courses exist with different creation timestamps
- **THEN** the system returns the more recently created course before the older one

### Requirement: Result count is bounded by a limit parameter
The system SHALL accept an optional `limit` parameter, applying a default and a maximum when it is omitted or exceeds the maximum.

#### Scenario: Explicit limit respected
- **WHEN** a client requests the trending list with `limit=5`
- **THEN** the system returns at most 5 items

#### Scenario: Omitted limit uses default
- **WHEN** a client requests the trending list without a `limit` parameter
- **THEN** the system returns the default number of items

#### Scenario: Excessive limit is clamped
- **WHEN** a client requests a `limit` above the documented maximum
- **THEN** the system clamps the result to the maximum instead of rejecting the request

### Requirement: Response is a published-only, public-safe projection
The system SHALL include only course id, slug, title, short description, thumbnail URL, instructor display name, and creation timestamp in each trending item, and SHALL NOT include draft-only, collaborator-only, pricing, rating/review, or review-workflow fields.

#### Scenario: No internal fields leak
- **WHEN** the trending list is returned
- **THEN** each item contains only the public-safe fields and no draft/collaborator/review-status data

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
- **THEN** the system still returns a correct trending list by querying the database directly
