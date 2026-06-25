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

type addRosterBulkRequest struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1,dive,uuid"`
}

type rosterBulkFailureResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type rosterBulkResponse struct {
	Added  []rosterResponse            `json:"added"`
	Failed []rosterBulkFailureResponse `json:"failed"`
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
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar"`
}

type rosterCandidateResponse struct {
	UserID       string `json:"user_id"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	AvatarFileID string `json:"avatar_file_id,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
}

type applicationResponse struct {
	ID              string      `json:"id"`
	UserID          string      `json:"user_id"`
	FullName        string      `json:"full_name"`
	AvatarURL       string      `json:"avatar"`
	ReviewStatus    string      `json:"review_status"`
	RejectionReason string      `json:"rejection_reason,omitempty"`
	Profile         profileBody `json:"profile"`
}

func toRosterResponse(m domain.RosterMember) rosterResponse {
	return rosterResponse{
		ID: m.UserID, FullName: m.FullName, Email: m.Email, Phone: m.Phone, AvatarURL: m.AvatarURL,
	}
}

func toRosterCandidateResponse(c domain.RosterCandidate) rosterCandidateResponse {
	return rosterCandidateResponse{
		UserID: c.UserID, DisplayName: c.DisplayName, Email: c.Email,
		AvatarFileID: c.AvatarFileID, AvatarURL: c.AvatarURL,
	}
}

func toRosterBulkResponse(result domain.RosterBulkResult) rosterBulkResponse {
	added := make([]rosterResponse, len(result.Added))
	for i, row := range result.Added {
		added[i] = toRosterResponse(row)
	}
	failed := make([]rosterBulkFailureResponse, len(result.Failed))
	for i, row := range result.Failed {
		failed[i] = rosterBulkFailureResponse{UserID: row.UserID, Message: row.Message}
	}
	return rosterBulkResponse{Added: added, Failed: failed}
}

func profileToResponse(row domain.Profile) applicationResponse {
	return applicationResponse{
		ID: row.ID, UserID: row.UserID, FullName: row.FullName, AvatarURL: row.AvatarURL, ReviewStatus: "managed",
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
		ID: a.ID, UserID: a.UserID, FullName: a.FullName, AvatarURL: a.AvatarURL,
		ReviewStatus: a.ReviewStatus, RejectionReason: a.RejectionReason,
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
	TopicID string `json:"topic_id" binding:"required,uuid"`
}

type expertiseSkillRequest struct {
	SkillID string `json:"skill_id" binding:"required,uuid"`
}

type createTicketRequest struct {
	Subject string `json:"subject" binding:"required"`
}

type ticketMessageRequest struct {
	Body string `json:"body" binding:"required"`
}
