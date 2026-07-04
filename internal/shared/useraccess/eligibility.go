package useraccess

import "errors"

var ErrEmailNotConfirmed = errors.New("email is not confirmed")

// AssignmentSnapshot extends Snapshot with fields required for role assignment.
type AssignmentSnapshot struct {
	Snapshot
	EmailConfirmed bool
}

// CheckEligibleForAssignment validates #2 (accessible) and #3 (email confirmed).
func CheckEligibleForAssignment(snapshot *AssignmentSnapshot, now int64) error {
	if snapshot == nil {
		return ErrUserNotFound
	}
	if err := CheckAccessible(&snapshot.Snapshot, now); err != nil {
		return err
	}
	if !snapshot.EmailConfirmed {
		return ErrEmailNotConfirmed
	}
	return nil
}

// AssignmentFailureMessage maps eligibility errors to bulk API failure messages.
func AssignmentFailureMessage(err error) string {
	switch {
	case errors.Is(err, ErrUserNotFound):
		return ErrUserNotFound.Error()
	case errors.Is(err, ErrUserDisabled):
		return ErrUserDisabled.Error()
	case errors.Is(err, ErrUserBanned):
		return ErrUserBanned.Error()
	case errors.Is(err, ErrEmailNotConfirmed):
		return ErrEmailNotConfirmed.Error()
	default:
		return "user is not eligible for assignment"
	}
}
