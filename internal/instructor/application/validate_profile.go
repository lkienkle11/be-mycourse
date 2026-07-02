package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
)

var validYearsCodes = map[string]struct{}{
	domain.YearsUnder1Year:       {},
	domain.YearsOneToTwoYears:    {},
	domain.YearsThreeToFiveYears: {},
	domain.YearsSixToTenYears:    {},
	domain.YearsOverTenYears:     {},
}

func (s *InstructorService) validateProfile(ctx context.Context, p domain.ProfilePayload) error {
	if s.mediaVal != nil {
		if err := s.mediaVal.ValidateProfilePayload(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (s *InstructorService) validateSubmitInput(ctx context.Context, in domain.SubmitApplicationInput) error {
	p := in.ProfilePayload
	if err := s.validateProfile(ctx, p); err != nil {
		return err
	}
	bio := strings.TrimSpace(p.Bio)
	if len(bio) < 100 || len(bio) > 2000 {
		return domain.ErrInvalidApplicationPayload
	}
	if _, ok := validYearsCodes[strings.TrimSpace(p.YearsOfExperience)]; !ok {
		return domain.ErrInvalidApplicationPayload
	}
	if strings.TrimSpace(p.CurrentJobTitleID) == "" {
		return domain.ErrInvalidApplicationPayload
	}
	if len(in.TopicIDs) < 1 || len(in.TopicIDs) > 5 {
		return domain.ErrInvalidApplicationPayload
	}
	if len(in.SkillIDs) < 1 || len(in.SkillIDs) > 15 {
		return domain.ErrInvalidApplicationPayload
	}
	if len(in.PortfolioLinks) > 5 {
		return domain.ErrInvalidApplicationPayload
	}
	if len(in.Certificates) > 10 {
		return domain.ErrInvalidApplicationPayload
	}
	return nil
}

func normalizeRejectionReason(reason string) (string, error) {
	r := strings.TrimSpace(reason)
	if len(r) < 1 || len(r) > 2000 {
		return "", domain.ErrRejectionReasonRequired
	}
	return r, nil
}
