package application

import (
	"context"
	"errors"

	"mycourse-io-be/internal/auth/domain"
	sharedErrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/useraccess"
)

func checkUserAccessible(user *domain.User, now int64) error {
	if user == nil {
		return domain.ErrUserNotFound
	}
	err := useraccess.CheckAccessible(&useraccess.Snapshot{
		DeletedAt:   user.DeletedAt,
		IsDisabled:  user.IsDisable,
		BannedUntil: user.BannedUntil,
	}, now)
	if err == nil {
		return nil
	}
	switch err {
	case useraccess.ErrUserNotFound:
		return domain.ErrUserNotFound
	case useraccess.ErrUserDisabled:
		return domain.ErrUserDisabled
	case useraccess.ErrUserBanned:
		return domain.ErrUserBanned
	default:
		return err
	}
}

// EnsureActiveUser verifies the user is not soft-deleted, disabled, or actively banned.
// Returns shared/errors sentinels for middleware.RequireActiveUser.
func (s *AuthService) EnsureActiveUser(ctx context.Context, userID string) error {
	_, err := s.loadAccessibleUser(ctx, userID)
	return toSharedAccessErr(err)
}

func toSharedAccessErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return sharedErrors.ErrUserNotFound
	case errors.Is(err, domain.ErrUserDisabled):
		return sharedErrors.ErrUserDisabled
	case errors.Is(err, domain.ErrUserBanned):
		return sharedErrors.ErrUserBanned
	default:
		return err
	}
}
