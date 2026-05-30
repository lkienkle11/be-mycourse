package application

import (
	"context"

	"mycourse-io-be/internal/instructor/domain"
)

func (s *InstructorService) ListProfiles(ctx context.Context, f domain.ProfileFilter) ([]domain.Profile, int64, error) {
	return s.repo.ListProfiles(ctx, f)
}

func (s *InstructorService) GetProfileByUserID(ctx context.Context, userID uint) (*domain.Profile, error) {
	return s.repo.GetProfileByUserID(ctx, userID)
}

func (s *InstructorService) UpsertProfile(ctx context.Context, in domain.UpsertProfileInput) (*domain.Profile, error) {
	if err := s.validateProfile(ctx, in.ProfilePayload); err != nil {
		return nil, err
	}
	return s.repo.UpsertProfile(ctx, in)
}

func (s *InstructorService) DeleteProfile(ctx context.Context, userID uint) error {
	return s.repo.DeleteProfileByUserID(ctx, userID)
}
