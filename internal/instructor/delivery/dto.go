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

func (q listQuery) getRosterPerPage() int {
	perPage := q.getPerPage()
	if perPage > 100 {
		return 100
	}
	return perPage
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

type submitApplicationBody struct {
	profileBody
	TopicIDs []string `json:"topic_ids" binding:"required,min=1,max=5,dive,uuid"`
	SkillIDs []string `json:"skill_ids" binding:"required,min=1,max=15,dive,uuid"`
}

func (b submitApplicationBody) toInput(actorUserID string) domain.SubmitApplicationInput {
	return domain.SubmitApplicationInput{
		ActorUserID: actorUserID, TopicIDs: b.TopicIDs, SkillIDs: b.SkillIDs,
		ProfilePayload: b.toPayload(),
	}
}

type profileBody struct {
	Headline                  string            `json:"headline"`
	Bio                       string            `json:"bio"`
	YearsOfExperience         string            `json:"years_of_experience"`
	CurrentJobTitle           string            `json:"current_job_title"`
	CurrentJobTitleID         string            `json:"current_job_title_id"`
	CurrentCompany            string            `json:"current_company"`
	CurrentCompanyID          *string           `json:"current_company_id,omitempty"`
	CurrentCompanyDomain      *string           `json:"current_company_domain,omitempty"`
	CurrentCompanyDescription *string           `json:"current_company_description,omitempty"`
	CurrentCompanyLocation    *string           `json:"current_company_location,omitempty"`
	CVFileID                  string            `json:"cv_file_id"`
	CVFile                    *mediaFileBody    `json:"cv_file,omitempty"`
	LinkedinURL               string            `json:"linkedin_url"`
	GithubURL                 string            `json:"github_url"`
	PortfolioLinks            []string          `json:"portfolio_links"`
	Certificates              []certificateBody `json:"certificates"`
	IntroVideoFileID          string            `json:"intro_video_file_id"`
	IntroVideoFile            *mediaFileBody    `json:"intro_video_file,omitempty"`
}

type mediaFileBody struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type certificateBody struct {
	Title             string         `json:"title"`
	Issuer            string         `json:"issuer"`
	IssuedYear        int            `json:"issued_year"`
	CredentialURL     string         `json:"credential_url"`
	CertificateFileID string         `json:"certificate_file_id,omitempty"`
	CertificateFile   *mediaFileBody `json:"certificate_file,omitempty"`
}

func (b profileBody) toPayload() domain.ProfilePayload {
	certs := make([]domain.Certificate, len(b.Certificates))
	for i, c := range b.Certificates {
		certs[i] = domain.Certificate{
			Title: c.Title, Issuer: c.Issuer, IssuedYear: c.IssuedYear,
			CredentialURL: c.CredentialURL, CertificateFileID: c.CertificateFileID,
		}
	}
	links := b.PortfolioLinks
	if links == nil {
		links = []string{}
	}
	return domain.ProfilePayload{
		Headline: b.Headline, Bio: b.Bio, YearsOfExperience: b.YearsOfExperience,
		CurrentJobTitle: b.CurrentJobTitle, CurrentJobTitleID: b.CurrentJobTitleID, CurrentCompany: b.CurrentCompany,
		CompanySnapshot: domain.CompanySnapshot{
			CurrentCompanyID: b.CurrentCompanyID, CurrentCompanyDomain: b.CurrentCompanyDomain,
			CurrentCompanyDescription: b.CurrentCompanyDescription, CurrentCompanyLocation: b.CurrentCompanyLocation,
		},
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

type latestSubmissionResponse struct {
	Profile  profileBody `json:"profile"`
	TopicIDs []string    `json:"topic_ids"`
	SkillIDs []string    `json:"skill_ids"`
}

type rejectionRecordResponse struct {
	RejectedAt          int64  `json:"rejected_at"`
	RejectedByUserID    string `json:"rejected_by_user_id"`
	ReviewerDisplayName string `json:"reviewer_display_name"`
	Reason              string `json:"reason"`
}

type taxonomyChipResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type applicationMeResponse struct {
	ID               string                    `json:"id"`
	UserID           string                    `json:"user_id"`
	DisplayName      string                    `json:"display_name"`
	Email            string                    `json:"email"`
	Avatar           string                    `json:"avatar"`
	IsDisabled       bool                      `json:"is_disabled"`
	EmailConfirmed   bool                      `json:"email_confirmed"`
	BannedUntil      *int64                    `json:"banned_until,omitempty"`
	IsBanned         bool                      `json:"is_banned"`
	ReviewStatus     string                    `json:"review_status"`
	CanResubmit      bool                      `json:"can_resubmit"`
	RejectionCount   int                       `json:"rejection_count"`
	RejectionReason  string                    `json:"rejection_reason,omitempty"`
	SubmittedAt      int64                     `json:"submitted_at"`
	ReviewDueAt      int64                     `json:"review_due_at"`
	ReturnedAt       *int64                    `json:"returned_at"`
	LatestSubmission latestSubmissionResponse  `json:"latest_submission"`
	RejectionHistory []rejectionRecordResponse `json:"rejection_history"`
	Topics           []taxonomyChipResponse    `json:"topics,omitempty"`
	Skills           []taxonomyChipResponse    `json:"skills,omitempty"`
}

type applicationResponse struct {
	applicationMeResponse
}

type contactAdminRequest struct {
	Subject string `json:"subject" binding:"required"`
	Message string `json:"message" binding:"required"`
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
	profile := profileBodyFromPayload(row.ProfilePayload)
	if row.CVFile != nil {
		profile.CVFile = mediaFileFromDomain(row.CVFile)
	}
	if row.IntroVideoFile != nil {
		profile.IntroVideoFile = mediaFileFromDomain(row.IntroVideoFile)
	}
	return applicationResponse{applicationMeResponse: applicationMeResponse{
		ID: row.ID, UserID: row.UserID, DisplayName: row.FullName, Email: row.Email, Avatar: row.AvatarURL,
		IsDisabled: row.IsDisabled, EmailConfirmed: row.EmailConfirmed,
		ReviewStatus:     "managed",
		LatestSubmission: latestSubmissionResponse{Profile: profile},
	}}
}

func profileBodyFromPayload(p domain.ProfilePayload) profileBody {
	return profileBody{
		Headline: p.Headline, Bio: p.Bio, YearsOfExperience: p.YearsOfExperience,
		CurrentJobTitle: p.CurrentJobTitle, CurrentJobTitleID: p.CurrentJobTitleID,
		CurrentCompany:   p.CurrentCompany,
		CurrentCompanyID: p.CurrentCompanyID, CurrentCompanyDomain: p.CurrentCompanyDomain,
		CurrentCompanyDescription: p.CurrentCompanyDescription, CurrentCompanyLocation: p.CurrentCompanyLocation,
		CVFileID: p.CVFileID, LinkedinURL: p.LinkedinURL, GithubURL: p.GithubURL,
		PortfolioLinks:   p.PortfolioLinks,
		Certificates:     certBodiesFromDomain(p.Certificates),
		IntroVideoFileID: p.IntroVideoFileID,
	}
}

func toApplicationMeResponse(a domain.Application) applicationMeResponse {
	profile := profileBodyFromPayload(a.ProfilePayload)
	if a.CVFile != nil {
		profile.CVFile = mediaFileFromDomain(a.CVFile)
	}
	if a.IntroVideoFile != nil {
		profile.IntroVideoFile = mediaFileFromDomain(a.IntroVideoFile)
	}
	history := make([]rejectionRecordResponse, len(a.RejectionHistory))
	for i, r := range a.RejectionHistory {
		history[i] = rejectionRecordResponse{
			RejectedAt: r.RejectedAt, RejectedByUserID: r.RejectedByUserID,
			ReviewerDisplayName: r.ReviewerDisplayName, Reason: r.Reason,
		}
	}
	displayName := a.DisplayName
	if displayName == "" {
		displayName = a.FullName
	}
	resp := applicationMeResponse{
		ID: a.ID, UserID: a.UserID, DisplayName: displayName, Email: a.Email, Avatar: a.AvatarURL,
		IsDisabled: a.IsDisabled, EmailConfirmed: a.EmailConfirmed, BannedUntil: a.BannedUntil, IsBanned: a.IsBanned,
		ReviewStatus: a.ReviewStatus, CanResubmit: a.CanResubmit(), RejectionCount: a.RejectionCount,
		RejectionReason: a.RejectionReason, SubmittedAt: a.SubmittedAt, ReviewDueAt: a.ReviewDueAt,
		ReturnedAt: a.ReturnedAt, RejectionHistory: history,
		LatestSubmission: latestSubmissionResponse{
			Profile: profile, TopicIDs: a.TopicIDs, SkillIDs: a.SkillIDs,
		},
	}
	if len(a.Topics) > 0 {
		resp.Topics = taxonomyChipsFromDomain(a.Topics)
	}
	if len(a.Skills) > 0 {
		resp.Skills = taxonomyChipsFromDomain(a.Skills)
	}
	return resp
}

func toApplicationResponse(a domain.Application) applicationResponse {
	return applicationResponse{applicationMeResponse: toApplicationMeResponse(a)}
}

func taxonomyChipsFromDomain(chips []domain.ApplicationTaxonomyChip) []taxonomyChipResponse {
	out := make([]taxonomyChipResponse, len(chips))
	for i, c := range chips {
		out[i] = taxonomyChipResponse{ID: c.ID, Name: c.Name, Slug: c.Slug}
	}
	return out
}

func mediaFileFromDomain(f *domain.MediaFileReadModel) *mediaFileBody {
	if f == nil {
		return nil
	}
	return &mediaFileBody{ID: f.ID, URL: f.URL, Filename: f.Filename, MimeType: f.MimeType}
}

func certBodiesFromDomain(certs []domain.Certificate) []certificateBody {
	out := make([]certificateBody, len(certs))
	for i, c := range certs {
		body := certificateBody{
			Title: c.Title, Issuer: c.Issuer, IssuedYear: c.IssuedYear,
			CredentialURL: c.CredentialURL, CertificateFileID: c.CertificateFileID,
		}
		if c.CertificateFile != nil {
			body.CertificateFile = mediaFileFromDomain(c.CertificateFile)
		}
		out[i] = body
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

type ticketResponse struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Avatar      string `json:"avatar"`
	Subject     string `json:"subject"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type ticketMessageResponse struct {
	ID             string `json:"id"`
	TicketID       string `json:"ticket_id"`
	AuthorUserID   string `json:"author_user_id"`
	AuthorFullName string `json:"author_full_name"`
	AuthorEmail    string `json:"author_email"`
	Body           string `json:"body"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

func toTicketResponse(t domain.Ticket) ticketResponse {
	return ticketResponse{
		ID: t.ID, UserID: t.UserID, DisplayName: t.DisplayName, Email: t.Email, Avatar: t.AvatarURL,
		Subject: t.Subject, Status: t.Status, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toTicketResponses(rows []domain.Ticket) []ticketResponse {
	out := make([]ticketResponse, len(rows))
	for i, row := range rows {
		out[i] = toTicketResponse(row)
	}
	return out
}

func toTicketMessageResponse(m domain.TicketMessage) ticketMessageResponse {
	return ticketMessageResponse{
		ID: m.ID, TicketID: m.TicketID, AuthorUserID: m.AuthorUserID,
		AuthorFullName: m.AuthorFullName, AuthorEmail: m.AuthorEmail,
		Body: m.Body, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toTicketMessageResponses(rows []domain.TicketMessage) []ticketMessageResponse {
	out := make([]ticketMessageResponse, len(rows))
	for i, row := range rows {
		out[i] = toTicketMessageResponse(row)
	}
	return out
}
