package taxonomy

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/logic/mapping"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/requestutil"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/pkg/validate"
	taxonomyservice "mycourse-io-be/services/taxonomy"
)

func listCourseLevels(c *gin.Context) {
	var q dto.CourseLevelFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}

	rows, total, err := taxonomyservice.ListCourseLevels(q)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return
	}

	response.OKPaginated(c, "ok", mapping.ToCourseLevelResponses(rows), utils.BuildPage(q.GetPage(), q.GetPerPage(), total))
}

func createCourseLevel(c *gin.Context) {
	var req dto.CreateCourseLevelRequest
	if err := validate.BindJSON(c, &req); err != nil {
		httperr.Abort(c, err)
		return
	}

	row, err := taxonomyservice.CreateCourseLevel(requestutil.CurrentUserID(c), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", mapping.ToCourseLevelResponse(*row))
}

func updateCourseLevel(c *gin.Context) {
	id, ok := requestutil.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}

	var req dto.UpdateCourseLevelRequest
	if err := validate.BindJSON(c, &req); err != nil {
		httperr.Abort(c, err)
		return
	}

	row, err := taxonomyservice.UpdateCourseLevel(id, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusNotFound, errcode.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", mapping.ToCourseLevelResponse(*row))
}

func deleteCourseLevel(c *gin.Context) {
	id, ok := requestutil.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid id", nil)
		return
	}

	if err := taxonomyservice.DeleteCourseLevel(id); err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}
