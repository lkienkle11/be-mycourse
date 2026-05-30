package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

func (s *InstructorService) ListRoster(ctx context.Context, f domain.RosterFilter) ([]domain.RosterMember, int64, error) {
	rows, total, err := s.repo.ListRoster(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	if err := s.hydrateRosterAvatars(ctx, rows); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *InstructorService) hydrateRosterAvatars(ctx context.Context, rows []domain.RosterMember) error {
	if s.hydrator == nil || len(rows) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		if id := strings.TrimSpace(r.AvatarFileID); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	urls, err := s.hydrator.ResolveAvatarURLs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range rows {
		if u, ok := urls[rows[i].AvatarFileID]; ok {
			rows[i].AvatarURL = u
		}
	}
	return nil
}

func (s *InstructorService) AddRosterByEmail(ctx context.Context, email string) (*domain.RosterMember, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, apperrors.ErrNotFound
	}
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if err := s.roles.AssignInstructorRole(ctx, user.ID); err != nil {
		return nil, err
	}
	s.invalidateMe(ctx, user.ID)
	member := domain.RosterMember{
		UserID: user.ID, FullName: user.DisplayName, Email: user.Email,
	}
	if user.AvatarFileID != nil {
		member.AvatarFileID = *user.AvatarFileID
	}
	_ = s.hydrateRosterAvatars(ctx, []domain.RosterMember{member})
	return &member, nil
}

func (s *InstructorService) RemoveFromRoster(ctx context.Context, userID uint) error {
	if err := s.repo.WipeInstructorScopedData(ctx, userID); err != nil {
		return err
	}
	if err := s.roles.RemoveInstructorRole(ctx, userID); err != nil {
		return err
	}
	s.invalidateMe(ctx, userID)
	return nil
}
