package application

import (
	"context"

	"mycourse-io-be/internal/instructor/domain"
)

// InstructorService implements instructor dashboard use cases.
type InstructorService struct {
	repo     domain.Repository
	users    UserLookup
	roles    InstructorRoleManager
	meCache  MeCacheInvalidator
	mediaVal ProfileMediaValidator
	hydrator AvatarHydrator
}

func NewInstructorService(
	repo domain.Repository,
	users UserLookup,
	roles InstructorRoleManager,
	meCache MeCacheInvalidator,
	mediaVal ProfileMediaValidator,
	hydrator AvatarHydrator,
) *InstructorService {
	return &InstructorService{
		repo: repo, users: users, roles: roles, meCache: meCache,
		mediaVal: mediaVal, hydrator: hydrator,
	}
}

func (s *InstructorService) invalidateMe(ctx context.Context, userID uint) {
	if s.meCache != nil && userID > 0 {
		s.meCache.InvalidateUserMeCache(ctx, userID)
	}
}
