package application

import (
	"context"
	stderrors "errors"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
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
	result := domain.RosterBulkResult{
		Added:  make([]domain.RosterMember, 0, len(userIDs)),
		Failed: make([]domain.RosterBulkFailure, 0),
	}
	seen := make(map[string]struct{}, len(userIDs))
	for _, rawID := range userIDs {
		userID := strings.TrimSpace(rawID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		member, err := s.AddRosterByUserID(ctx, userID)
		if err != nil {
			if msg, ok := rosterBulkClientMessage(err); ok {
				result.Failed = append(result.Failed, domain.RosterBulkFailure{
					UserID: userID, Message: msg,
				})
				continue
			}
			return domain.RosterBulkResult{}, err
		}
		result.Added = append(result.Added, *member)
	}
	return result, nil
}

// rosterBulkClientMessage maps known business errors to safe client messages.
// Returns false for unexpected/infrastructure errors (caller should abort with 500).
func rosterBulkClientMessage(err error) (string, bool) {
	if stderrors.Is(err, apperrors.ErrNotFound) {
		return "user not found", true
	}
	if stderrors.Is(err, domain.ErrRosterPlatformStaffUser) {
		return domain.ErrRosterPlatformStaffUser.Error(), true
	}
	return "", false
}

func (s *InstructorService) AddRosterByUserID(ctx context.Context, userID string) (*domain.RosterMember, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, apperrors.ErrNotFound
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	staff, err := s.repo.UserHasPlatformStaffRole(ctx, userID)
	if err != nil {
		return nil, err
	}
	if staff {
		return nil, domain.ErrRosterPlatformStaffUser
	}
	return s.addRosterForUser(ctx, user.ID, user.DisplayName, user.Email, user.AvatarFileID)
}

func (s *InstructorService) addRosterForUser(ctx context.Context, userID, fullName, email string, avatarFileID *string) (*domain.RosterMember, error) {
	if err := s.roles.AssignInstructorRole(ctx, userID); err != nil {
		return nil, err
	}
	s.invalidateMe(ctx, userID)
	member := domain.RosterMember{
		UserID: userID, FullName: fullName, Email: email,
	}
	if avatarFileID != nil {
		member.AvatarFileID = *avatarFileID
	}
	_ = hydrateAvatarURLsByAccessor(ctx, s.hydrator, []domain.RosterMember{member},
		rosterMemberAvatarAccessor.getFileID, rosterMemberAvatarAccessor.setURL)
	return &member, nil
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
