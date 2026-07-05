package application

import (
	"context"
	"strconv"
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
	if err := validateSubmitProfileFields(p, in.TopicIDs, in.SkillIDs); err != nil {
		return err
	}
	if err := validateProfileLinkURLs(p); err != nil {
		return err
	}
	if err := validateCertificatePayload(p.Certificates); err != nil {
		return err
	}
	return nil
}

func validateSubmitProfileFields(p domain.ProfilePayload, topicIDs, skillIDs []string) error {
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
	if strings.TrimSpace(p.CurrentJobTitle) == "" {
		return domain.ErrInvalidApplicationPayload
	}
	if strings.TrimSpace(p.CurrentCompany) == "" {
		return domain.ErrInvalidApplicationPayload
	}
	if strings.TrimSpace(p.CVFileID) == "" {
		return domain.ErrInvalidApplicationPayload
	}
	if len(topicIDs) < 1 || len(topicIDs) > 5 {
		return domain.ErrInvalidApplicationPayload
	}
	if len(skillIDs) < 1 || len(skillIDs) > 15 {
		return domain.ErrInvalidApplicationPayload
	}
	if len(p.PortfolioLinks) > 5 {
		return domain.ErrInvalidApplicationPayload
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
	seen := certificateSeenSet{
		composite: make(map[string]struct{}, len(certs)),
		urls:      make(map[string]struct{}, len(certs)),
		fileIDs:   make(map[string]struct{}, len(certs)),
	}
	for _, cert := range certs {
		title := strings.TrimSpace(cert.Title)
		if title == "" {
			if certificateRowHasPartialData(cert) {
				return domain.ErrInvalidApplicationPayload
			}
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
		if seen.register(title, cert.Issuer, cert.IssuedYear, url, fileID) {
			return domain.ErrDuplicateCertificate
		}
	}
	return nil
}

// certificateSeenSet tracks distinct certificate identities within one payload
// to reject duplicate rows (same composite, same credential URL, or same file).
type certificateSeenSet struct {
	composite map[string]struct{}
	urls      map[string]struct{}
	fileIDs   map[string]struct{}
}

func (s *certificateSeenSet) register(title, issuer string, year int, url, fileID string) bool {
	composite := sharedutils.NormalizeDedupeKey(title) + "|" + sharedutils.NormalizeDedupeKey(issuer) + "|" + strconv.Itoa(year)
	dup := s.mark(s.composite, composite)
	if url != "" {
		dup = s.mark(s.urls, url) || dup
	}
	if fileID != "" {
		dup = s.mark(s.fileIDs, fileID) || dup
	}
	return dup
}

func (s *certificateSeenSet) mark(set map[string]struct{}, key string) bool {
	if _, ok := set[key]; ok {
		return true
	}
	set[key] = struct{}{}
	return false
}

func certificateRowHasPartialData(cert domain.Certificate) bool {
	if strings.TrimSpace(cert.Issuer) != "" {
		return true
	}
	if strings.TrimSpace(cert.CredentialURL) != "" {
		return true
	}
	if strings.TrimSpace(cert.CertificateFileID) != "" {
		return true
	}
	return false
}

func normalizeRejectionReason(reason string) (string, error) {
	r := strings.TrimSpace(reason)
	if len(r) < 1 || len(r) > 2000 {
		return "", domain.ErrRejectionReasonRequired
	}
	return r, nil
}
