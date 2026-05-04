package constants

// MaxActiveSessions is the maximum number of concurrent device refresh sessions per user.
// When a new session is created at the cap, the session with the earliest expiry is evicted first.
const MaxActiveSessions = 5
