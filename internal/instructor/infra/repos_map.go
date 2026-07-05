package infra

import (
	"errors"

	"gorm.io/gorm"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrNotFound
	}
	return err
}

type profileFields struct {
	Headline, Bio, YearsOfExperience, CurrentJobTitle, CurrentJobTitleID, CurrentCompany      string
	CurrentCompanyID, CurrentCompanyDomain, CurrentCompanyDescription, CurrentCompanyLocation *string
	CVFileID, LinkedinURL, GithubURL, IntroVideoFileID                                        string
	PortfolioLinks                                                                            *StringSliceJSON
	Certificates                                                                              *CertificatesJSON
}

func nullableStringPtr(s string) *string {
	s = trimNullable(s)
	if s == "" {
		return nil
	}
	return &s
}

func trimNullable(s string) string {
	// strings import avoided in map file — use inline trim
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\t' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func fieldsFromPayload(p domain.ProfilePayload) (profileFields, error) {
	pl, cj, err := payloadToJSONFields(p)
	if err != nil {
		return profileFields{}, err
	}
	return profileFields{
		Headline: p.Headline, Bio: p.Bio, YearsOfExperience: p.YearsOfExperience,
		CurrentJobTitle: p.CurrentJobTitle, CurrentJobTitleID: p.CurrentJobTitleID,
		CurrentCompany:            p.CurrentCompany,
		CurrentCompanyID:          nullableStringPtr(derefString(p.CurrentCompanyID)),
		CurrentCompanyDomain:      nullableStringPtr(derefString(p.CurrentCompanyDomain)),
		CurrentCompanyDescription: nullableStringPtr(derefString(p.CurrentCompanyDescription)),
		CurrentCompanyLocation:    nullableStringPtr(derefString(p.CurrentCompanyLocation)),
		CVFileID:                  p.CVFileID, LinkedinURL: p.LinkedinURL, GithubURL: p.GithubURL,
		IntroVideoFileID: p.IntroVideoFileID, PortfolioLinks: &pl, Certificates: &cj,
	}, nil
}

func payloadFromFields(f profileFields) domain.ProfilePayload {
	return domain.ProfilePayload{
		Headline: f.Headline, Bio: f.Bio, YearsOfExperience: f.YearsOfExperience,
		CurrentJobTitle: f.CurrentJobTitle, CurrentJobTitleID: f.CurrentJobTitleID,
		CurrentCompany: f.CurrentCompany,
		CompanySnapshot: domain.CompanySnapshot{
			CurrentCompanyID:          f.CurrentCompanyID,
			CurrentCompanyDomain:      f.CurrentCompanyDomain,
			CurrentCompanyDescription: f.CurrentCompanyDescription,
			CurrentCompanyLocation:    f.CurrentCompanyLocation,
		},
		CVFileID: f.CVFileID, LinkedinURL: f.LinkedinURL, GithubURL: f.GithubURL,
		IntroVideoFileID: f.IntroVideoFileID,
		PortfolioLinks:   stringSliceFromJSON(f.PortfolioLinks),
		Certificates:     certificatesFromJSON(certSliceFromJSON(f.Certificates)),
	}
}

func rowFieldsFromData(d *ProfileDataRow) profileFields {
	return profileFields{
		Headline: d.Headline, Bio: d.Bio, YearsOfExperience: d.YearsOfExperience,
		CurrentJobTitle: d.CurrentJobTitle, CurrentJobTitleID: d.CurrentJobTitleID,
		CurrentCompany:   d.CurrentCompany,
		CurrentCompanyID: d.CurrentCompanyID, CurrentCompanyDomain: d.CurrentCompanyDomain,
		CurrentCompanyDescription: d.CurrentCompanyDescription, CurrentCompanyLocation: d.CurrentCompanyLocation,
		CVFileID: d.CVFileID, LinkedinURL: d.LinkedinURL, GithubURL: d.GithubURL,
		IntroVideoFileID: d.IntroVideoFileID, PortfolioLinks: d.PortfolioLinks, Certificates: d.Certificates,
	}
}

func writeFieldsToData(d *ProfileDataRow, f profileFields) {
	d.Headline, d.Bio, d.YearsOfExperience = f.Headline, f.Bio, f.YearsOfExperience
	d.CurrentJobTitle, d.CurrentJobTitleID, d.CurrentCompany = f.CurrentJobTitle, f.CurrentJobTitleID, f.CurrentCompany
	d.CurrentCompanyID, d.CurrentCompanyDomain = f.CurrentCompanyID, f.CurrentCompanyDomain
	d.CurrentCompanyDescription, d.CurrentCompanyLocation = f.CurrentCompanyDescription, f.CurrentCompanyLocation
	d.CVFileID, d.LinkedinURL, d.GithubURL = f.CVFileID, f.LinkedinURL, f.GithubURL
	d.PortfolioLinks, d.Certificates, d.IntroVideoFileID = f.PortfolioLinks, f.Certificates, f.IntroVideoFileID
}

func rejectionHistoryFromJSON(h *RejectionHistoryJSON) []domain.RejectionRecord {
	if h == nil {
		return []domain.RejectionRecord{}
	}
	raw := []rejectionHistoryJSON(*h)
	out := make([]domain.RejectionRecord, len(raw))
	for i, r := range raw {
		out[i] = domain.RejectionRecord{
			RejectedAt: r.RejectedAt, RejectedByUserID: r.RejectedByUserID,
			ReviewerDisplayName: r.ReviewerDisplayName, Reason: r.Reason,
		}
	}
	return out
}

func rejectionHistoryToJSON(records []domain.RejectionRecord) RejectionHistoryJSON {
	out := make(RejectionHistoryJSON, len(records))
	for i, r := range records {
		out[i] = rejectionHistoryJSON{
			RejectedAt: r.RejectedAt, RejectedByUserID: r.RejectedByUserID,
			ReviewerDisplayName: r.ReviewerDisplayName, Reason: r.Reason,
		}
	}
	return out
}

func appRowToDomain(r *applicationRow) domain.Application {
	history := rejectionHistoryFromJSON(r.RejectionHistory)
	return domain.Application{
		ID: r.ID, UserID: r.UserID, ReviewStatus: r.ReviewStatus,
		RejectionReason: r.RejectionReason, SubmittedAt: r.SubmittedAt,
		ReviewDueAt: r.ReviewDueAt, ReturnedAt: r.ReturnedAt,
		RejectionCount: r.RejectionCount, RejectionHistory: history,
		ProfilePayload: payloadFromFields(rowFieldsFromData(&r.ProfileDataRow)),
		CreatedAt:      r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func profileRowToDomain(r *profileRow) domain.Profile {
	return domain.Profile{
		ID: r.ID, UserID: r.UserID, ProfilePayload: payloadFromFields(rowFieldsFromData(&r.ProfileDataRow)),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func expertiseTopicRowToDomain(r *expertiseTopicRow, name, slug string) domain.ExpertiseTopic {
	return domain.ExpertiseTopic{
		ID: r.ID, UserID: r.UserID, TopicID: r.TopicID,
		Name: name, Slug: slug,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func expertiseSkillRowToDomain(r *expertiseSkillRow, name, slug string) domain.ExpertiseSkill {
	return domain.ExpertiseSkill{
		ID: r.ID, UserID: r.UserID, SkillID: r.SkillID,
		Name: name, Slug: slug,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func applyPayloadToAppRow(r *applicationRow, p domain.ProfilePayload) error {
	f, err := fieldsFromPayload(p)
	if err != nil {
		return err
	}
	writeFieldsToData(&r.ProfileDataRow, f)
	return nil
}

func applyPayloadToProfileRow(r *profileRow, p domain.ProfilePayload) error {
	f, err := fieldsFromPayload(p)
	if err != nil {
		return err
	}
	writeFieldsToData(&r.ProfileDataRow, f)
	return nil
}

func stringSliceFromJSON(s *StringSliceJSON) []string {
	if s == nil {
		return []string{}
	}
	return []string(*s)
}

func certSliceFromJSON(c *CertificatesJSON) []certificateJSON {
	if c == nil {
		return []certificateJSON{}
	}
	return []certificateJSON(*c)
}

func payloadToJSONFields(p domain.ProfilePayload) (StringSliceJSON, CertificatesJSON, error) {
	pl := StringSliceJSON(append([]string(nil), p.PortfolioLinks...))
	cj, err := certificatesToJSON(p.Certificates)
	if err != nil {
		return nil, nil, err
	}
	return pl, CertificatesJSON(cj), nil
}
