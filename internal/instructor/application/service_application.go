package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/useraccess"
)

func (s *InstructorService) ListApplications(ctx context.Context, f domain.ApplicationFilter) ([]domain.Application, int64, error) {
	rows, total, err := listWithIdentity(
		s,
		ctx,
		func() ([]domain.Application, int64, error) { return s.repo.ListApplications(ctx, f) },
		applicationAvatarFileID,
		setApplicationAvatarURL,
	)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		if e := s.enrichApplication(ctx, &rows[i], false); e != nil {
			return nil, 0, e
		}
	}
	return rows, total, nil
}

func (s *InstructorService) GetApplication(ctx context.Context, id string) (*domain.Application, error) {
	row, err := loadOneWithIdentity(
		s,
		ctx,
		func() (*domain.Application, error) { return s.repo.GetApplicationByID(ctx, id) },
		applicationAvatarFileID,
		setApplicationAvatarURL,
	)
	if err != nil {
		return nil, err
	}
	if err := s.enrichApplication(ctx, row, true); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *InstructorService) GetMyApplication(ctx context.Context, userID string) (*domain.Application, error) {
	if err := s.repo.MarkReturnedIfDue(ctx, userID); err != nil {
		return nil, err
	}
	row, err := loadOneWithIdentity(
		s,
		ctx,
		func() (*domain.Application, error) { return s.repo.GetActiveApplicationByUserID(ctx, userID) },
		applicationAvatarFileID,
		setApplicationAvatarURL,
	)
	if err != nil {
		return nil, err
	}
	if err := s.enrichApplication(ctx, row, true); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *InstructorService) SubmitApplication(ctx context.Context, in domain.SubmitApplicationInput) (*domain.Application, error) {
	return s.submitApplication(ctx, in, s.repo.CreateFirstApplication)
}

func (s *InstructorService) ResubmitMyApplication(ctx context.Context, in domain.SubmitApplicationInput) (*domain.Application, error) {
	return s.submitApplication(ctx, in, s.repo.ResubmitApplication)
}

type applicationPersistFn func(context.Context, string, domain.SubmitApplicationInput) (*domain.Application, error)

func (s *InstructorService) submitApplication(ctx context.Context, in domain.SubmitApplicationInput, persist applicationPersistFn) (*domain.Application, error) {
	if err := s.assertCanSubmit(ctx, in.ActorUserID); err != nil {
		return nil, err
	}
	if err := s.validateSubmitInput(ctx, in); err != nil {
		return nil, err
	}
	row, err := persist(ctx, in.ActorUserID, in)
	if err != nil {
		return nil, err
	}
	return s.hydrateApplicationResponse(ctx, row)
}

func (s *InstructorService) ApproveApplication(ctx context.Context, id string) (*domain.Application, error) {
	app, err := s.repo.GetApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if app.ReviewStatus != domain.ReviewStatusPending {
		return nil, domain.ErrApplicationNotPending
	}
	if err := s.assertApplicantEligibleForReview(ctx, app.UserID); err != nil {
		return nil, err
	}
	// DB first: copy snapshot + set approved in one transaction; role grant only after success.
	if err := s.repo.ApproveApplicationCopySnapshot(ctx, id, app.UserID); err != nil {
		return nil, err
	}
	if err := s.roles.AssignInstructorRole(ctx, app.UserID); err != nil {
		return nil, err
	}
	s.invalidateMe(ctx, app.UserID)
	return s.GetApplication(ctx, id)
}

func (s *InstructorService) RejectApplication(ctx context.Context, in domain.RejectApplicationInput) (*domain.Application, error) {
	reason, err := normalizeRejectionReason(in.RejectionReason)
	if err != nil {
		return nil, err
	}
	in.RejectionReason = reason
	app, err := s.repo.GetApplicationByID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app.ReviewStatus != domain.ReviewStatusPending {
		return nil, domain.ErrApplicationNotPending
	}
	if err := s.assertApplicantEligibleForReview(ctx, app.UserID); err != nil {
		return nil, err
	}
	if in.ReviewerDisplayName == "" && s.users != nil && in.ReviewerUserID != "" {
		if u, uerr := s.users.FindByID(ctx, in.ReviewerUserID); uerr == nil && u != nil {
			in.ReviewerDisplayName = u.DisplayName
		}
	}
	if err := s.repo.RejectApplicationWithHistory(ctx, in); err != nil {
		return nil, err
	}
	return s.GetApplication(ctx, in.ApplicationID)
}

func (s *InstructorService) assertApplicantEligibleForReview(ctx context.Context, userID string) error {
	if s.users == nil {
		return nil
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	snap := useraccess.AssignmentSnapshot{
		Snapshot: useraccess.Snapshot{
			DeletedAt: u.DeletedAt, IsDisabled: u.IsDisable, BannedUntil: u.BannedUntil,
		},
		EmailConfirmed: u.EmailConfirmed,
	}
	return useraccess.CheckEligibleForAssignment(&snap, timex.NowUnix())
}

func (s *InstructorService) DeleteApplication(ctx context.Context, id string) error {
	app, err := s.repo.GetApplicationByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.DeleteApplicationsByUserID(ctx, app.UserID)
}

func (s *InstructorService) ApplicationHasProfile(p domain.ProfilePayload) bool {
	return strings.TrimSpace(p.CVFileID) != ""
}

func (s *InstructorService) assertCanSubmit(ctx context.Context, userID string) error {
	if s.perms != nil && s.perms.HasPermission(ctx, userID, constants.AllPermissions.InstructorApplicationSubmitBlocked) {
		return domain.ErrApplicationSubmitBlocked
	}
	return nil
}

func (s *InstructorService) enrichApplication(ctx context.Context, app *domain.Application, withDetail bool) error {
	topicIDs, err := s.repo.ListApplicationTopicIDs(ctx, app.ID)
	if err != nil {
		return err
	}
	skillIDs, err := s.repo.ListApplicationSkillIDs(ctx, app.ID)
	if err != nil {
		return err
	}
	app.TopicIDs = topicIDs
	app.SkillIDs = skillIDs
	if withDetail {
		topics, err := s.repo.ListApplicationTopics(ctx, app.ID)
		if err != nil {
			return err
		}
		skills, err := s.repo.ListApplicationSkills(ctx, app.ID)
		if err != nil {
			return err
		}
		app.Topics = topics
		app.Skills = skills
		if err := s.hydrateApplicationMedia(ctx, app); err != nil {
			return err
		}
	}
	return nil
}

func (s *InstructorService) hydrateApplicationResponse(ctx context.Context, app *domain.Application) (*domain.Application, error) {
	if err := s.enrichApplication(ctx, app, true); err != nil {
		return nil, err
	}
	items := []domain.Application{*app}
	if err := hydrateAvatarURLsByAccessor(ctx, s.hydrator, items, applicationAvatarFileID, setApplicationAvatarURL); err != nil {
		return nil, err
	}
	*app = items[0]
	return app, nil
}

func (s *InstructorService) hydrateApplicationMedia(ctx context.Context, app *domain.Application) error {
	return hydrateProfilePayloadMedia(
		ctx,
		s.mediaHydr,
		&app.ProfilePayload,
		func(f *domain.MediaFileReadModel) { app.CVFile = f },
		func(f *domain.MediaFileReadModel) { app.IntroVideoFile = f },
	)
}

// CreateContactTicket creates an instructor support ticket for State H contact flow.
func (s *InstructorService) CreateContactTicket(ctx context.Context, userID, subject, body string) (*domain.Ticket, error) {
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	if subject == "" || body == "" {
		return nil, domain.ErrInvalidApplicationPayload
	}
	app, err := s.repo.GetActiveApplicationByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if app.RejectionCount < domain.MaxApplicationRejections {
		return nil, domain.ErrApplicationContactNotAllowed
	}
	ticket, err := s.repo.CreateTicketWithFirstMessage(ctx, userID, subject, body)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}
