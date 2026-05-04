package constants

import "time"

// Auth JWT / refresh lifetimes (shared with services and api cookie Max-Age).
const (
	AccessTokenTTL       = 15 * time.Minute
	RefreshTokenTTL      = 30 * 24 * time.Hour // default / non-remember-me initial TTL
	RememberMeRefreshTTL = 14 * 24 * time.Hour // remember-me: renewed to this on every rotation
)
