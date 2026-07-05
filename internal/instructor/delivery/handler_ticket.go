package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
)

func (h *Handler) listTickets(c *gin.Context) {
	var q listQuery
	_ = c.ShouldBindQuery(&q)
	f := domain.TicketFilter{Page: q.getPage(), PageSize: q.getPerPage(), Status: q.Status}
	if c.Query("scope") != "all" {
		f.UserID = utils.CurrentUserID(c)
	}
	rows, total, err := h.svc.ListTickets(c.Request.Context(), f)
	if mapInstructorError(c, err) {
		return
	}
	response.OKPaginated(c, "ok", toTicketResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) createTicket(c *gin.Context) {
	var req createTicketRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.CreateTicket(c.Request.Context(), utils.CurrentUserID(c), req.Subject)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", toTicketResponse(*row))
}

func (h *Handler) closeTicket(c *gin.Context) {
	h.withTicketID(c, func(id string) (any, error) {
		row, err := h.svc.CloseTicket(c.Request.Context(), id)
		if err != nil {
			return nil, err
		}
		return toTicketResponse(*row), nil
	})
}

func (h *Handler) listTicketMessages(c *gin.Context) {
	h.withTicketID(c, func(id string) (any, error) {
		rows, err := h.svc.ListTicketMessages(c.Request.Context(), id)
		if err != nil {
			return nil, err
		}
		return toTicketMessageResponses(rows), nil
	})
}

func (h *Handler) withTicketID(c *gin.Context, fn func(string) (any, error)) {
	id, ok := parseIDParam(c)
	if !ok {
		failInvalidID(c)
		return
	}
	out, err := fn(id)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", out)
}

func (h *Handler) addTicketMessage(c *gin.Context) {
	var req ticketMessageRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	h.withTicketID(c, func(id string) (any, error) {
		row, err := h.svc.AddTicketMessage(c.Request.Context(), id, utils.CurrentUserID(c), req.Body)
		if err != nil {
			return nil, err
		}
		return toTicketMessageResponse(*row), nil
	})
}
