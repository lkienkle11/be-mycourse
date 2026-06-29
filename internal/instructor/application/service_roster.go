package application

import (
	"context"

	"mycourse-io-be/internal/instructor/domain"
	sharedutils "mycourse-io-be/internal/shared/utils"
)

type avatarAccessor[T any] struct {
	getFileID func(T) string
	setURL    func(*T, string)
}

var (
	rosterMemberAvatarAccessor = avatarAccessor[domain.RosterMember]{
		getFileID: func(row domain.RosterMember) string { return row.AvatarFileID },
		setURL:    func(row *domain.RosterMember, url string) { row.AvatarURL = url },
	}
)

func listRepoWithAvatarHydrate[F any, T any](
	ctx context.Context,
	filter F,
	list func(context.Context, F) ([]T, int64, error),
	hydrator AvatarHydrator,
	accessor avatarAccessor[T],
) ([]T, int64, error) {
	rows, total, err := list(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if err := hydrateAvatarURLsByAccessor(ctx, hydrator, rows, accessor.getFileID, accessor.setURL); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *InstructorService) ListRoster(ctx context.Context, f domain.RosterFilter) ([]domain.RosterMember, int64, error) {
	return listRepoWithAvatarHydrate(ctx, f, s.repo.ListRoster, s.hydrator, rosterMemberAvatarAccessor)
}

func (s *InstructorService) ListRosterCandidates(ctx context.Context, f domain.RosterCandidateFilter) ([]domain.RosterCandidate, int64, error) {
	return s.repo.ListRosterCandidates(ctx, f)
}

func (s *InstructorService) AddRosterBulk(ctx context.Context, userIDs []string) (domain.RosterBulkResult, error) {
	prepared := sharedutils.PrepareBulkUserIDs(userIDs)
	if len(prepared) == 0 {
		return domain.RosterBulkResult{
			Added:  []domain.RosterMember{},
			Failed: []domain.RosterBulkFailure{},
		}, nil
	}
	roleID, err := s.roles.InstructorRoleID(ctx)
	if err != nil {
		return domain.RosterBulkResult{}, err
	}
	result, err := s.repo.AddRosterBulk(ctx, prepared, roleID)
	if err != nil {
		return domain.RosterBulkResult{}, err
	}
	for _, userID := range result.InsertedUserIDs {
		s.invalidateMe(ctx, userID)
	}
	_ = hydrateAvatarURLsByAccessor(ctx, s.hydrator, result.Added,
		rosterMemberAvatarAccessor.getFileID, rosterMemberAvatarAccessor.setURL)
	return result, nil
}

func (s *InstructorService) RemoveFromRoster(ctx context.Context, userID string) error {
	if err := s.repo.WipeInstructorScopedData(ctx, userID); err != nil {
		return err
	}
	if err := s.roles.RemoveInstructorRole(ctx, userID); err != nil {
		return err
	}
	s.invalidateMe(ctx, userID)
	return nil
}
