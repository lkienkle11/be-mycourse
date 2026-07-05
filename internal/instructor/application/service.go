package application

import (
	"context"

	"mycourse-io-be/internal/instructor/domain"
)

// InstructorService implements instructor dashboard use cases.
type InstructorService struct {
	repo                domain.Repository
	users               UserLookup
	roles               InstructorRoleManager
	perms               PermissionChecker
	meCache             MeCacheInvalidator
	mediaVal            ProfileMediaValidator
	hydrator            AvatarHydrator
	mediaHydr           MediaHydrator
	assignmentSnapshots AssignmentSnapshotLoader
}

// InstructorServiceDeps groups optional instructor service collaborators.
type InstructorServiceDeps struct {
	Users               UserLookup
	Roles               InstructorRoleManager
	Perms               PermissionChecker
	MeCache             MeCacheInvalidator
	MediaVal            ProfileMediaValidator
	Hydrator            AvatarHydrator
	MediaHydr           MediaHydrator
	AssignmentSnapshots AssignmentSnapshotLoader
}

func NewInstructorService(repo domain.Repository, deps InstructorServiceDeps) *InstructorService {
	return &InstructorService{
		repo: repo, users: deps.Users, roles: deps.Roles, perms: deps.Perms, meCache: deps.MeCache,
		mediaVal: deps.MediaVal, hydrator: deps.Hydrator, mediaHydr: deps.MediaHydr,
		assignmentSnapshots: deps.AssignmentSnapshots,
	}
}

func (s *InstructorService) invalidateMe(ctx context.Context, userID string) {
	if s.meCache != nil && userID != "" {
		s.meCache.InvalidateUserMeCache(ctx, userID)
	}
}
