package application

import (
	"context"

	authdomain "mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/instructor/domain"
)

// UserLookup resolves application users by email or id.
type UserLookup interface {
	FindByEmail(ctx context.Context, email string) (*authdomain.User, error)
	FindByID(ctx context.Context, userID string) (*authdomain.User, error)
}

// InstructorRoleManager assigns or removes the instructor role only.
type InstructorRoleManager interface {
	InstructorRoleID(ctx context.Context) (uint, error)
	AssignInstructorRole(ctx context.Context, userID string) error
	RemoveInstructorRole(ctx context.Context, userID string) error
	UserHasInstructorRole(ctx context.Context, userID string) (bool, error)
}

// PermissionChecker resolves effective permissions for submit-block rules.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, action string) bool
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

// MediaHydrator resolves full media read models for CV and intro video.
type MediaHydrator interface {
	ResolveMediaFiles(ctx context.Context, fileIDs []string) (map[string]domain.MediaFileReadModel, error)
}
