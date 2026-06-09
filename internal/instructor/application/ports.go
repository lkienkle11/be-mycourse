package application

import (
	"context"

	authdomain "mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/instructor/domain"
)

// UserLookup resolves application users by email.
type UserLookup interface {
	FindByEmail(ctx context.Context, email string) (*authdomain.User, error)
}

// InstructorRoleManager assigns or removes the instructor role only.
type InstructorRoleManager interface {
	InstructorRoleID(ctx context.Context) (uint, error)
	AssignInstructorRole(ctx context.Context, userID string) error
	RemoveInstructorRole(ctx context.Context, userID string) error
}

// MeCacheInvalidator clears cached /me after RBAC changes.
type MeCacheInvalidator interface {
	InvalidateUserMeCache(ctx context.Context, userID string)
}

// ProfileMediaValidator checks media file IDs on profile payloads.
type ProfileMediaValidator interface {
	ValidateProfilePayload(ctx context.Context, p domain.ProfilePayload) error
}

// AvatarHydrator maps media file IDs to public URLs.
type AvatarHydrator interface {
	ResolveAvatarURLs(ctx context.Context, fileIDs []string) (map[string]string, error)
}
