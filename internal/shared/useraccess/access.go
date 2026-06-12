package useraccess

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserDisabled = errors.New("user account is disabled")
	ErrUserBanned   = errors.New("user account is temporarily banned")
)

// Snapshot contains the minimal user fields needed for accessibility checks.
type Snapshot struct {
	DeletedAt   *int64
	IsDisabled  bool
	BannedUntil *int64
}

// CheckAccessible returns a sentinel error when the user cannot be accessed.
func CheckAccessible(snapshot *Snapshot, now int64) error {
	if snapshot == nil || snapshot.DeletedAt != nil {
		return ErrUserNotFound
	}
	if snapshot.IsDisabled {
		return ErrUserDisabled
	}
	if snapshot.BannedUntil != nil && *snapshot.BannedUntil > now {
		return ErrUserBanned
	}
	return nil
}
