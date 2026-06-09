package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
)

func (h *Handler) deleteByID(c *gin.Context, fn func(string) error) {
	id, ok := parseIDParam(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if mapInstructorError(c, fn(id)) {
		return
	}
	response.OK(c, "deleted", nil)
}
