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
	if stderrors.Is(err, domain.ErrApplicationNotResubmittable) ||
		stderrors.Is(err, domain.ErrApplicationAlreadyExists) ||
		stderrors.Is(err, domain.ErrApplicationSubmitBlocked) ||
		stderrors.Is(err, domain.ErrApplicationRejectQuota) ||
		stderrors.Is(err, domain.ErrApplicationAlreadyInstructor) ||
		stderrors.Is(err, domain.ErrApplicationContactNotAllowed) ||
		stderrors.Is(err, domain.ErrInvalidApplicationPayload) {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return true
	}
	if stderrors.Is(err, domain.ErrTicketClosed) {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return true
	}
	if stderrors.Is(err, domain.ErrRosterPlatformStaffUser) {
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

func parseIDParam(c *gin.Context) (string, bool) {
	return utils.ParseUUIDParam(c, "id")
}

func failInvalidID(c *gin.Context) {
	response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
}

func listPaginatedWithQuery[T any, R any](
	c *gin.Context,
	list func(context.Context, listQuery) ([]T, int64, error),
	mapRows func([]T) []R,
) {
	httpx.ListPaginated(c,
		func(q *listQuery) error { return c.ShouldBindQuery(q) },
		list,
		func(q listQuery) (int, int) { return q.getPage(), q.getPerPage() },
		mapRows,
	)
}

func (h *Handler) respondApplicationByID(c *gin.Context, load func(context.Context, string) (*domain.Application, error)) {
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
