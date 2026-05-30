package delivery

import (
	"mycourse-io-be/internal/instructor/domain"
)

type listQuery struct {
	Page       int    `form:"page"`
	PerPage    int    `form:"per_page"`
	Search     string `form:"search"`
	Status     string `form:"status"`
	HasProfile *bool  `form:"has_profile"`
}

func (q listQuery) getPage() int {
	if q.Page < 1 {
		return 1
	}
	return q.Page
}

func (q listQuery) getPerPage() int {
	if q.PerPage < 1 {
		return 20
	}
	return q.PerPage
}

type addRosterRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type rejectApplicationRequest struct {
	RejectionReason string `json:"rejection_reason" binding:"required"`
}

type profileBody struct {
	Headline          string            `json:"headline"`
	Bio               string            `json:"bio"`
	YearsOfExperience int               `json:"years_of_experience"`
	CurrentJobTitle   string            `json:"current_job_title"`
	CurrentCompany    string            `json:"current_company"`
	CVFileID          string            `json:"cv_file_id"`
	LinkedinURL       string            `json:"linkedin_url"`
	GithubURL         string            `json:"github_url"`
	PortfolioLinks    []string          `json:"portfolio_links"`
	Certificates      []certificateBody `json:"certificates"`
	IntroVideoFileID  string            `json:"intro_video_file_id"`
}

type certificateBody struct {
	Title         string `json:"title"`
	Issuer        string `json:"issuer"`
	IssuedYear    int    `json:"issued_year"`
	CredentialURL string `json:"credential_url"`
}

func (b profileBody) toPayload() domain.ProfilePayload {
	certs := make([]domain.Certificate, len(b.Certificates))
	for i, c := range b.Certificates {
		certs[i] = domain.Certificate{
			Title: c.Title, Issuer: c.Issuer, IssuedYear: c.IssuedYear, CredentialURL: c.CredentialURL,
		}
	}
	links := b.PortfolioLinks
	if links == nil {
		links = []string{}
	}
	return domain.ProfilePayload{
		Headline: b.Headline, Bio: b.Bio, YearsOfExperience: b.YearsOfExperience,
		CurrentJobTitle: b.CurrentJobTitle, CurrentCompany: b.CurrentCompany,
		CVFileID: b.CVFileID, LinkedinURL: b.LinkedinURL, GithubURL: b.GithubURL,
		PortfolioLinks: links, Certificates: certs, IntroVideoFileID: b.IntroVideoFileID,
	}
}

type rosterResponse struct {
	ID        uint   `json:"id"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar"`
}

type applicationResponse struct {
	ID              uint        `json:"id"`
	UserID          uint        `json:"user_id"`
	ReviewStatus    string      `json:"review_status"`
	RejectionReason string      `json:"rejection_reason,omitempty"`
	Profile         profileBody `json:"profile"`
}

func toRosterResponse(m domain.RosterMember) rosterResponse {
	return rosterResponse{
		ID: m.UserID, FullName: m.FullName, Email: m.Email, Phone: m.Phone, AvatarURL: m.AvatarURL,
	}
}

func profileToResponse(row domain.Profile) applicationResponse {
	return applicationResponse{
		ID: row.ID, UserID: row.UserID, ReviewStatus: "managed",
		Profile: profileBodyFromPayload(row.ProfilePayload),
	}
}

func profileBodyFromPayload(p domain.ProfilePayload) profileBody {
	return profileBody{
		Headline: p.Headline, Bio: p.Bio, YearsOfExperience: p.YearsOfExperience,
		CurrentJobTitle: p.CurrentJobTitle, CurrentCompany: p.CurrentCompany,
		CVFileID: p.CVFileID, LinkedinURL: p.LinkedinURL, GithubURL: p.GithubURL,
		PortfolioLinks:   p.PortfolioLinks,
		Certificates:     certBodiesFromDomain(p.Certificates),
		IntroVideoFileID: p.IntroVideoFileID,
	}
}

func toApplicationResponse(a domain.Application) applicationResponse {
	p := a.ProfilePayload
	return applicationResponse{
		ID: a.ID, UserID: a.UserID, ReviewStatus: a.ReviewStatus, RejectionReason: a.RejectionReason,
		Profile: profileBodyFromPayload(p),
	}
}

func certBodiesFromDomain(certs []domain.Certificate) []certificateBody {
	out := make([]certificateBody, len(certs))
	for i, c := range certs {
		out[i] = certificateBody{
			Title: c.Title, Issuer: c.Issuer, IssuedYear: c.IssuedYear, CredentialURL: c.CredentialURL,
		}
	}
	return out
}

type expertiseTopicRequest struct {
	TopicID uint `json:"topic_id" binding:"required"`
}

type expertiseSkillRequest struct {
	SkillID uint `json:"skill_id" binding:"required"`
}

type createTicketRequest struct {
	Subject string `json:"subject" binding:"required"`
}

type ticketMessageRequest struct {
	Body string `json:"body" binding:"required"`
}
