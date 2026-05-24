package application

import (
	"context"
	"errors"

	"mycourse-io-be/internal/auth/domain"
	sharedErrors "mycourse-io-be/internal/shared/errors"
)

func checkUserAccessible(user *domain.User, now int64) error {
	if user == nil {
		return domain.ErrUserNotFound
	}
	if user.DeletedAt != nil {
		return domain.ErrUserNotFound
	}
	if user.IsDisable {
		return domain.ErrUserDisabled
	}
	if isUserBanned(now, user.BannedUntil) {
		return domain.ErrUserBanned
	}
	return nil
}

func isUserBanned(now int64, bannedUntil *int64) bool {
	return bannedUntil != nil && *bannedUntil > now
}

// EnsureActiveUser verifies the user is not soft-deleted, disabled, or actively banned.
// Returns shared/errors sentinels for middleware.RequireActiveUser.
func (s *AuthService) EnsureActiveUser(ctx context.Context, userID uint) error {
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
