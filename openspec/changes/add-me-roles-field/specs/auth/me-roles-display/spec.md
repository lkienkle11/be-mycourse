## Purpose

Surfaces a user's RBAC role names on the `/me` profile endpoint purely for client-side display (e.g. showing a job title/chức danh), without becoming part of any authorization decision.

## ADDED Requirements

### Requirement: `/me` returns display-only role names
The `GET /api/v1/me` response SHALL include a `roles` field: an array of the authenticated user's RBAC role names, in raw lowercase English form exactly as stored (e.g. `sysadmin`, `admin`, `instructor`, `learner`), intended only for client-side display and never for authorization decisions.

#### Scenario: User with a single role
- **WHEN** an authenticated user with exactly one RBAC role binding (`instructor`) calls `GET /api/v1/me`
- **THEN** the response includes `"roles": ["instructor"]`

#### Scenario: User with multiple roles
- **WHEN** an authenticated user holds both the `admin` and `instructor` role bindings and calls `GET /api/v1/me`
- **THEN** the response includes both role names in `roles`

#### Scenario: User with no role bindings
- **WHEN** an authenticated user has no RBAC role bindings at all and calls `GET /api/v1/me`
- **THEN** the response includes `"roles": []` (an empty array, never `null` and never an omitted field)

### Requirement: Roles are ordered by a fixed priority ranking
When a user holds more than one role, the `roles` array SHALL be ordered by a fixed, hardcoded priority ranking from highest to lowest: `sysadmin`, `admin`, `instructor`, `learner`. A role name not present in the ranking SHALL be placed after every ranked role name, preserving a stable order among unranked names.

#### Scenario: Higher-privilege role sorts first
- **WHEN** a user holds both `instructor` and `admin` role bindings
- **THEN** the response orders `roles` as `["admin", "instructor"]`, not `["instructor", "admin"]`

#### Scenario: Role name outside the fixed ranking
- **WHEN** a user holds a role binding whose name is not one of `sysadmin`, `admin`, `instructor`, `learner`
- **THEN** that role name still appears in `roles`, placed after all ranked role names

### Requirement: `roles` is excluded from authorization logic
The `roles` field on `/me` SHALL NOT be consulted by any backend authorization check, permission gate, or middleware, and SHALL NOT be documented or treated as a substitute for permission codes. Access-control decisions SHALL continue to rely exclusively on permission codes (as returned by `GET /api/v1/me/permissions` and evaluated via the existing permission-check path).

#### Scenario: Role present but permission absent
- **WHEN** a user's `roles` array contains `admin` but the user's permission codes do not include the permission required for a given action
- **THEN** the action SHALL be denied, exactly as it would be denied for a user with no roles at all

### Requirement: Cached `/me` responses stay consistent with current role bindings
The `/me` response caching mechanism SHALL be invalidated whenever a role or direct permission is assigned to or removed from a user, so that a subsequent `GET /api/v1/me` call reflects the user's current RBAC projection rather than stale cached `roles` or `permissions`. A cached `/me` entry written before this capability existed (lacking a `roles` field) SHALL NOT cause the endpoint to error; it SHALL be treated as having no cached roles and refreshed. Cache invalidation SHALL NOT be described as revoking permissions already embedded in an issued access token; those claims change only after refresh or expiry.

#### Scenario: Role assignment invalidates cache
- **WHEN** an admin assigns a new role to a user who has a cached `/me` entry
- **THEN** the next `GET /api/v1/me` call from that user returns a `roles` array that includes the newly assigned role

#### Scenario: Direct permission removal invalidates cache
- **WHEN** an admin removes a direct permission from a user who has a cached `/me` entry
- **THEN** the next `GET /api/v1/me` call reflects the current effective permission set without waiting for cache TTL
- **AND** an access token issued before the removal retains its immutable claims until refresh or expiry

#### Scenario: Pre-existing cache entry without roles
- **WHEN** a cached `/me` entry created before this capability was deployed is read back (no `roles` field present in the cached payload)
- **THEN** the endpoint responds successfully, treating the cached entry as stale and refreshing it rather than failing to deserialize
