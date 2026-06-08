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
	Headline, Bio, CurrentJobTitle, CurrentCompany, CVFileID, LinkedinURL, GithubURL, IntroVideoFileID string
	YearsOfExperience                                                                                  int
	PortfolioLinks                                                                                     *StringSliceJSON
	Certificates                                                                                       *CertificatesJSON
}

func fieldsFromPayload(p domain.ProfilePayload) (profileFields, error) {
	pl, cj, err := payloadToJSONFields(p)
	if err != nil {
		return profileFields{}, err
	}
	return profileFields{
		Headline: p.Headline, Bio: p.Bio, YearsOfExperience: p.YearsOfExperience,
		CurrentJobTitle: p.CurrentJobTitle, CurrentCompany: p.CurrentCompany,
		CVFileID: p.CVFileID, LinkedinURL: p.LinkedinURL, GithubURL: p.GithubURL,
		IntroVideoFileID: p.IntroVideoFileID, PortfolioLinks: &pl, Certificates: &cj,
	}, nil
}

func payloadFromFields(f profileFields) domain.ProfilePayload {
	return domain.ProfilePayload{
		Headline: f.Headline, Bio: f.Bio, YearsOfExperience: f.YearsOfExperience,
		CurrentJobTitle: f.CurrentJobTitle, CurrentCompany: f.CurrentCompany,
		CVFileID: f.CVFileID, LinkedinURL: f.LinkedinURL, GithubURL: f.GithubURL,
		IntroVideoFileID: f.IntroVideoFileID,
		PortfolioLinks:   stringSliceFromJSON(f.PortfolioLinks),
		Certificates:     certificatesFromJSON(certSliceFromJSON(f.Certificates)),
	}
}

func rowFieldsFromData(d *profileDataRow) profileFields {
	return profileFields{
		Headline: d.Headline, Bio: d.Bio, YearsOfExperience: d.YearsOfExperience,
		CurrentJobTitle: d.CurrentJobTitle, CurrentCompany: d.CurrentCompany,
		CVFileID: d.CVFileID, LinkedinURL: d.LinkedinURL, GithubURL: d.GithubURL,
		IntroVideoFileID: d.IntroVideoFileID, PortfolioLinks: d.PortfolioLinks, Certificates: d.Certificates,
	}
}

func writeFieldsToData(d *profileDataRow, f profileFields) {
	d.Headline, d.Bio, d.YearsOfExperience = f.Headline, f.Bio, f.YearsOfExperience
	d.CurrentJobTitle, d.CurrentCompany = f.CurrentJobTitle, f.CurrentCompany
	d.CVFileID, d.LinkedinURL, d.GithubURL = f.CVFileID, f.LinkedinURL, f.GithubURL
	d.PortfolioLinks, d.Certificates, d.IntroVideoFileID = f.PortfolioLinks, f.Certificates, f.IntroVideoFileID
}

func appRowToDomain(r *applicationRow) domain.Application {
	return domain.Application{
		ID: r.ID, UserID: r.UserID, ReviewStatus: r.ReviewStatus,
		RejectionReason: r.RejectionReason, ProfilePayload: payloadFromFields(rowFieldsFromData(&r.profileDataRow)),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func profileRowToDomain(r *profileRow) domain.Profile {
	return domain.Profile{
		ID: r.ID, UserID: r.UserID, ProfilePayload: payloadFromFields(rowFieldsFromData(&r.profileDataRow)),
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
	writeFieldsToData(&r.profileDataRow, f)
	return nil
}

func applyPayloadToProfileRow(r *profileRow, p domain.ProfilePayload) error {
	f, err := fieldsFromPayload(p)
	if err != nil {
		return err
	}
	writeFieldsToData(&r.profileDataRow, f)
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
