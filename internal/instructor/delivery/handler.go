package delivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/instructor/application"
	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/httpx"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
)

// Handler serves instructor dashboard HTTP APIs.
type Handler struct {
	svc *application.InstructorService
}

func NewHandler(svc *application.InstructorService) *Handler { return &Handler{svc: svc} }

func (h *Handler) listRoster(c *gin.Context) {
	httpx.ListPaginated(c,
		func(q *listQuery) error { return c.ShouldBindQuery(q) },
		func(ctx context.Context, q listQuery) ([]domain.RosterMember, int64, error) {
			return h.svc.ListRoster(ctx, domain.RosterFilter{Page: q.getPage(), PageSize: q.getPerPage(), Search: q.Search})
		},
		func(q listQuery) (int, int) { return q.getPage(), q.getPerPage() },
		func(rows []domain.RosterMember) []rosterResponse {
			out := make([]rosterResponse, len(rows))
			for i, r := range rows {
				out[i] = toRosterResponse(r)
			}
			return out
		},
	)
}

func (h *Handler) addRoster(c *gin.Context) {
	var req addRosterRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.AddRosterByEmail(c.Request.Context(), req.Email)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", toRosterResponse(*row))
}

func (h *Handler) deleteRoster(c *gin.Context) {
	id, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.RemoveFromRoster(c.Request.Context(), id); mapInstructorError(c, err) {
		return
	}
	response.OK(c, "deleted", nil)
}

func (h *Handler) listApplications(c *gin.Context) {
	httpx.ListPaginated(c,
		func(q *listQuery) error { return c.ShouldBindQuery(q) },
		func(ctx context.Context, q listQuery) ([]domain.Application, int64, error) {
			return h.svc.ListApplications(ctx, domain.ApplicationFilter{
				Page: q.getPage(), PageSize: q.getPerPage(), HasProfile: q.HasProfile, ReviewStatus: q.Status,
			})
		},
		func(q listQuery) (int, int) { return q.getPage(), q.getPerPage() },
		func(rows []domain.Application) []applicationResponse {
			out := make([]applicationResponse, len(rows))
			for i, r := range rows {
				out[i] = toApplicationResponse(r)
			}
			return out
		},
	)
}

func (h *Handler) getApplication(c *gin.Context) { h.respondApplicationByID(c, h.svc.GetApplication) }

func (h *Handler) submitApplication(c *gin.Context) {
	var req profileBody
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.SubmitApplication(c.Request.Context(), domain.SubmitApplicationInput{
		ActorUserID: utils.CurrentUserID(c), ProfilePayload: req.toPayload(),
	})
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", toApplicationResponse(*row))
}

func (h *Handler) approveApplication(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	row, err := h.svc.ApproveApplication(c.Request.Context(), id)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", toApplicationResponse(*row))
}

func (h *Handler) rejectApplication(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req rejectApplicationRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.RejectApplication(c.Request.Context(), domain.RejectApplicationInput{
		ApplicationID: id, RejectionReason: req.RejectionReason,
	})
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", toApplicationResponse(*row))
}

func (h *Handler) deleteApplication(c *gin.Context) {
	h.deleteByID(c, func(id string) error {
		return h.svc.DeleteApplication(c.Request.Context(), id)
	})
}

func (h *Handler) comingSoon(c *gin.Context) {
	response.OK(c, "coming soon", gin.H{"status": "coming_soon"})
}
