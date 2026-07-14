package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
)

func (h *Handler) listExpertiseTopics(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	locale := c.Query("locale")
	rows, err := h.svc.ListExpertiseTopics(c.Request.Context(), userID, locale)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

func (h *Handler) addExpertiseTopic(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	var req expertiseTopicRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.AddExpertiseTopic(c.Request.Context(), userID, req.TopicID)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", row)
}

func (h *Handler) deleteExpertiseTopicByRow(c *gin.Context) {
	id, ok := utils.ParseUUIDPathParam(c, "topicRowId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if mapInstructorError(c, h.svc.DeleteExpertiseTopic(c.Request.Context(), id)) {
		return
	}
	response.OK(c, "deleted", nil)
}

func parseUserIDParam(c *gin.Context) (string, bool) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return "", false
	}
	return id, true
}
