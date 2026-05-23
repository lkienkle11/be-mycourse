package httpx

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

// ListPaginated binds query params, runs listFn, and writes a paginated OK response.
// Returns false when the handler has already written an error response.
func ListPaginated[TQuery any, TRow any, TResp any](
	c *gin.Context,
	bind func(*TQuery) error,
	listFn func(context.Context, TQuery) ([]TRow, int64, error),
	pageOf func(TQuery) (page, perPage int),
	toResponses func([]TRow) []TResp,
) bool {
	var q TQuery
	if err := bind(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return false
	}
	rows, total, err := listFn(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return false
	}
	page, perPage := pageOf(q)
	response.OKPaginated(c, "ok", toResponses(rows), utils.BuildPage(page, perPage, total))
	return true
}
