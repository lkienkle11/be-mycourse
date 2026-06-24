package domain

import "time"

// Auth JWT / refresh lifetimes (shared with services and API cookie Max-Age).
const (
	AccessTokenTTL       = 15 * time.Minute
	RefreshTokenTTL      = 3 * 24 * time.Hour  // default / non-remember-me / email-confirm initial TTL
	RememberMeRefreshTTL = 30 * 24 * time.Hour // remember-me: renewed to this on every rotation
)
