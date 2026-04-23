package taxonomy

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/requestutil"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/pkg/validate"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listTags(c *gin.Context) {
	var q dto.TagFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	rows, total, err := taxonomyservice.ListTags(q)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return
	}

	perPage := q.GetPerPage()
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages == 0 {
		totalPages = 1
	}

	response.OKPaginated(c, "ok", rows, response.PageInfo{
		Page:       q.GetPage(),
		PerPage:    perPage,
		TotalPages: totalPages,
		TotalItems: int(total),
	})
}

func createTag(c *gin.Context) {
	var req dto.CreateTagRequest
	if err := validate.BindJSON(c, &req); err != nil {
		httperr.Abort(c, err)
		return
	}

	row, err := taxonomyservice.CreateTag(requestutil.CurrentUserID(c), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", row)
}

func updateTag(c *gin.Context) {
	id, ok := requestutil.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}

	var req dto.UpdateTagRequest
	if err := validate.BindJSON(c, &req); err != nil {
		httperr.Abort(c, err)
		return
	}

	row, err := taxonomyservice.UpdateTag(id, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", row)
}

func deleteTag(c *gin.Context) {
	id, ok := requestutil.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}

	if err := taxonomyservice.DeleteTag(id); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}
