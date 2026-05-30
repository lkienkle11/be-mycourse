package delivery

import (
	"context"
	stderrors "errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/httpx"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func mapInstructorError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if stderrors.Is(err, apperrors.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
		return true
	}
	if stderrors.Is(err, domain.ErrRejectionReasonRequired) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return true
	}
	if stderrors.Is(err, domain.ErrApplicationNotPending) {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return true
	}
	if stderrors.Is(err, domain.ErrTicketClosed) {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrInvalidProfileMediaFile) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return true
	}
	response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
	return true
}

func listPaginated[TRow any, TResp any](
	c *gin.Context,
	listFn func(context.Context, listQuery) ([]TRow, int64, error),
	toResp func([]TRow) []TResp,
) {
	httpx.ListPaginated(c,
		func(q *listQuery) error { return c.ShouldBindQuery(q) },
		func(ctx context.Context, q listQuery) ([]TRow, int64, error) { return listFn(ctx, q) },
		func(q listQuery) (int, int) { return q.getPage(), q.getPerPage() },
		toResp,
	)
}

func parseIDParam(c *gin.Context) (uint, bool) {
	return utils.ParseUintParam(c, "id")
}

func failInvalidID(c *gin.Context) {
	response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
}

func (h *Handler) respondApplicationByID(c *gin.Context, load func(context.Context, uint) (*domain.Application, error)) {
	id, ok := parseIDParam(c)
	if !ok {
		failInvalidID(c)
		return
	}
	row, err := load(c.Request.Context(), id)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", toApplicationResponse(*row))
}

func (h *Handler) respondProfileMe(c *gin.Context) {
	row, err := h.svc.GetProfileByUserID(c.Request.Context(), utils.CurrentUserID(c))
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", profileToResponse(*row))
}
