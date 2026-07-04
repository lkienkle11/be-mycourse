package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
	sharedutils "mycourse-io-be/internal/shared/utils"
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
	if strings.TrimSpace(p.CVFileID) == "" {
		return domain.ErrInvalidApplicationPayload
	}
	if len(in.TopicIDs) < 1 || len(in.TopicIDs) > 5 {
		return domain.ErrInvalidApplicationPayload
	}
	if len(in.SkillIDs) < 1 || len(in.SkillIDs) > 15 {
		return domain.ErrInvalidApplicationPayload
	}
	if len(p.PortfolioLinks) > 5 {
		return domain.ErrInvalidApplicationPayload
	}
	if err := validateProfileLinkURLs(p); err != nil {
		return err
	}
	if err := validateCertificatePayload(p.Certificates); err != nil {
		return err
	}
	return nil
}

func validateProfileLinkURLs(p domain.ProfilePayload) error {
	if !sharedutils.IsOptionalLinkedInURL(p.LinkedinURL) {
		return domain.ErrInvalidApplicationPayload
	}
	if !sharedutils.IsOptionalGitHubURL(p.GithubURL) {
		return domain.ErrInvalidApplicationPayload
	}
	for _, link := range p.PortfolioLinks {
		if !sharedutils.IsOptionalHTTPURL(link) {
			return domain.ErrInvalidApplicationPayload
		}
	}
	return nil
}

func validateCertificatePayload(certs []domain.Certificate) error {
	if len(certs) > 10 {
		return domain.ErrInvalidApplicationPayload
	}
	for _, cert := range certs {
		title := strings.TrimSpace(cert.Title)
		if title == "" {
			continue
		}
		if strings.TrimSpace(cert.Issuer) == "" || cert.IssuedYear < 1950 {
			return domain.ErrInvalidApplicationPayload
		}
		url := strings.TrimSpace(cert.CredentialURL)
		fileID := strings.TrimSpace(cert.CertificateFileID)
		if url == "" && fileID == "" {
			return domain.ErrInvalidApplicationPayload
		}
		if url != "" && !sharedutils.IsOptionalHTTPURL(url) {
			return domain.ErrInvalidApplicationPayload
		}
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
