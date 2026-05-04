package taxonomy

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/requestutil"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/pkg/validate"
)

func respondTaxonomyList[F dto.TaxonomyListFilter, M any](
	c *gin.Context,
	list func(F) ([]M, int64, error),
	mapRows func([]M) any,
) {
	var q F
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := list(q)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", mapRows(rows), utils.BuildPage(q.GetPage(), q.GetPerPage(), total))
}

func respondTaxonomyCreate[Req any, Row any](
	c *gin.Context,
	create func(uint, Req) (*Row, error),
	mapRow func(Row) any,
) {
	var req Req
	if err := validate.BindJSON(c, &req); err != nil {
		httperr.Abort(c, err)
		return
	}
	row, err := create(requestutil.CurrentUserID(c), req)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrInvalidProfileMediaFile) {
			response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", mapRow(*row))
}

func respondTaxonomyUpdate[Req any, Row any](
	c *gin.Context,
	update func(uint, Req) (*Row, error),
	mapRow func(Row) any,
) {
	id, ok := requestutil.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	var req Req
	if err := validate.BindJSON(c, &req); err != nil {
		httperr.Abort(c, err)
		return
	}
	row, err := update(id, req)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		if errors.Is(err, pkgerrors.ErrInvalidProfileMediaFile) {
			response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", mapRow(*row))
}

func respondTaxonomyDelete(c *gin.Context, del func(uint) error) {
	id, ok := requestutil.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}
	if err := del(id); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}
