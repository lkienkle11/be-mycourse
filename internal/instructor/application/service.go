package application

import (
	"context"

	"mycourse-io-be/internal/instructor/domain"
)

// InstructorService implements instructor dashboard use cases.
type InstructorService struct {
	repo      domain.Repository
	users     UserLookup
	roles     InstructorRoleManager
	perms     PermissionChecker
	meCache   MeCacheInvalidator
	mediaVal  ProfileMediaValidator
	hydrator  AvatarHydrator
	mediaHydr MediaHydrator
}

func NewInstructorService(
	repo domain.Repository,
	users UserLookup,
	roles InstructorRoleManager,
	perms PermissionChecker,
	meCache MeCacheInvalidator,
	mediaVal ProfileMediaValidator,
	hydrator AvatarHydrator,
	mediaHydr MediaHydrator,
) *InstructorService {
	return &InstructorService{
		repo: repo, users: users, roles: roles, perms: perms, meCache: meCache,
		mediaVal: mediaVal, hydrator: hydrator, mediaHydr: mediaHydr,
	}
}

func (s *InstructorService) invalidateMe(ctx context.Context, userID string) {
	if s.meCache != nil && userID != "" {
		s.meCache.InvalidateUserMeCache(ctx, userID)
	}
}
