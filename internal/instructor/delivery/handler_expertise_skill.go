package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
)

func (h *Handler) listExpertiseSkills(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	rows, err := h.svc.ListExpertiseSkills(c.Request.Context(), userID)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

func (h *Handler) addExpertiseSkill(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	var req expertiseSkillRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.AddExpertiseSkill(c.Request.Context(), userID, req.SkillID)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", row)
}

func (h *Handler) deleteExpertiseSkillByRow(c *gin.Context) {
	id, ok := utils.ParseUUIDPathParam(c, "skillRowId")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if mapInstructorError(c, h.svc.DeleteExpertiseSkill(c.Request.Context(), id)) {
		return
	}
	response.OK(c, "deleted", nil)
}
