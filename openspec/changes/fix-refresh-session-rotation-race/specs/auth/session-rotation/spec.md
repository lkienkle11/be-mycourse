## Purpose

Guarantees that refresh-token rotation has exactly one winner per refresh token and that no concurrent login, logout or user save can silently overwrite another device's valid session.

## ADDED Requirements

### Requirement: Refresh rotation has a single winner
The system SHALL replace a session's refresh token only when the stored refresh UUID of that session still equals the UUID carried by the presented refresh token. When several refresh requests present the same refresh token concurrently, exactly one SHALL succeed and the others SHALL fail as an invalid session.

#### Scenario: Two overlapping refreshes with the same token
- **WHEN** two `POST /api/v1/auth/refresh` requests with the same refresh token and session id are processed at the same time
- **THEN** exactly one responds 200 with a new token pair and the other responds 401 with code `InvalidSession`
- **AND** the stored refresh UUID of that session equals the UUID inside the refresh token of the 200 response

#### Scenario: Refresh after the token was already rotated
- **WHEN** a refresh request presents a refresh token whose UUID is no longer the stored UUID of the session
- **THEN** the system responds 401 with code `InvalidSession` and does not change the stored session

#### Scenario: Refresh after logout
- **WHEN** the session was removed (for example by logout) between the validity check and the rotation write
- **THEN** the rotation does not re-create the session and the request responds 401 with code `InvalidSession`

### Requirement: A lost rotation race is a session error, not a server error
The system SHALL answer a refresh request that lost the rotation race with the same status and code as any other invalid-session refresh. It SHALL NOT answer 500, and it SHALL NOT introduce a 403 reason that is not about the account state.

#### Scenario: Losing request status
- **WHEN** a refresh request loses the rotation race
- **THEN** the HTTP status is 401 and the error code is `InvalidSession`

#### Scenario: Unrelated statuses unchanged
- **WHEN** a refresh request has missing credentials, a disabled or banned user, an expired refresh token, or an internal failure
- **THEN** the status and code are the same as before this change (400, 403 `UserDisabled`/`UserBanned`, 401 `RefreshTokenExpired`, 500)

### Requirement: Session writes do not discard concurrent session changes
The system SHALL persist adding a session, removing a session and saving user profile fields without overwriting session entries written by a concurrent operation on the same user.

#### Scenario: Login while another device refreshes
- **WHEN** a login adds a session to a user while another session of the same user is being rotated
- **THEN** after both finish, the rotated session keeps its new refresh UUID and the new session is present (subject to the maximum of 5 sessions per user)

#### Scenario: Logout while another device refreshes
- **WHEN** a logout removes one session while a different session of the same user is being rotated
- **THEN** only the logged-out session is removed and the rotated session keeps its new refresh UUID

#### Scenario: Idempotent logout
- **WHEN** logout is requested for a session key that no longer exists
- **THEN** the request still succeeds and no other session changes

#### Scenario: User profile save does not touch sessions
- **WHEN** register-resend, email confirmation or an OAuth identity link saves the user's profile fields from an in-memory copy loaded earlier
- **THEN** the stored session map is not modified by that save

### Requirement: Existing session rules are preserved
The system SHALL keep the existing session rules: at most 5 concurrent sessions per user with eviction of the entry that expires first, rotation keeps the same session id, `remember_me` and the remaining lifetime rules are unchanged, and token and cookie formats are unchanged.

#### Scenario: Sixth login evicts the oldest entry
- **WHEN** a user with 5 active sessions logs in again
- **THEN** the session with the earliest `refresh_token_expired` is evicted and the new session is stored

#### Scenario: Successful rotation keeps the session id
- **WHEN** a refresh succeeds
- **THEN** the response carries the same session id, a new refresh UUID, and the same `remember_me` value
