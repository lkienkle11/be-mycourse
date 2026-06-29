package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
)

func (s *InstructorService) ListApplications(ctx context.Context, f domain.ApplicationFilter) ([]domain.Application, int64, error) {
	return listWithIdentity(
		s,
		ctx,
		func() ([]domain.Application, int64, error) { return s.repo.ListApplications(ctx, f) },
		applicationAvatarFileID,
		setApplicationAvatarURL,
	)
}

func (s *InstructorService) GetApplication(ctx context.Context, id string) (*domain.Application, error) {
	return loadOneWithIdentity(
		s,
		ctx,
		func() (*domain.Application, error) { return s.repo.GetApplicationByID(ctx, id) },
		applicationAvatarFileID,
		setApplicationAvatarURL,
	)
}

func (s *InstructorService) SubmitApplication(ctx context.Context, in domain.SubmitApplicationInput) (*domain.Application, error) {
	if err := s.validateProfile(ctx, in.ProfilePayload); err != nil {
		return nil, err
	}
	return loadOneWithIdentity(
		s,
		ctx,
		func() (*domain.Application, error) {
			return s.repo.UpsertPendingApplication(ctx, in.ActorUserID, in.ProfilePayload)
		},
		applicationAvatarFileID,
		setApplicationAvatarURL,
	)
}

func (s *InstructorService) ApproveApplication(ctx context.Context, id string) (*domain.Application, error) {
	app, err := s.repo.GetApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if app.ReviewStatus != domain.ReviewStatusPending {
		return nil, domain.ErrApplicationNotPending
	}
	if err := s.roles.AssignInstructorRole(ctx, app.UserID); err != nil {
		return nil, err
	}
	s.invalidateMe(ctx, app.UserID)
	if err := s.repo.SetApplicationReview(ctx, id, domain.ReviewStatusApproved, ""); err != nil {
		return nil, err
	}
	return s.GetApplication(ctx, id)
}

func (s *InstructorService) RejectApplication(ctx context.Context, in domain.RejectApplicationInput) (*domain.Application, error) {
	reason, err := normalizeRejectionReason(in.RejectionReason)
	if err != nil {
		return nil, err
	}
	app, err := s.repo.GetApplicationByID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app.ReviewStatus != domain.ReviewStatusPending {
		return nil, domain.ErrApplicationNotPending
	}
	if err := s.repo.SetApplicationReview(ctx, in.ApplicationID, domain.ReviewStatusRejected, reason); err != nil {
		return nil, err
	}
	return s.GetApplication(ctx, in.ApplicationID)
}

func (s *InstructorService) DeleteApplication(ctx context.Context, id string) error {
	app, err := s.repo.GetApplicationByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.DeleteApplicationsByUserID(ctx, app.UserID)
}

func (s *InstructorService) ApplicationHasProfile(p domain.ProfilePayload) bool {
	return strings.TrimSpace(p.Headline) != "" && strings.TrimSpace(p.CVFileID) != ""
}
