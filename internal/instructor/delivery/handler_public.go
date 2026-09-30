package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

const (
	popularInstructorsDefaultLimit = 4
	popularInstructorsMaxLimit     = 12
)

func (h *Handler) listPopularInstructors(c *gin.Context) {
	limit := utils.ClampQueryLimit(c.DefaultQuery("limit", ""), popularInstructorsDefaultLimit, popularInstructorsMaxLimit)
	rows, err := h.svc.ListPopularInstructors(c.Request.Context(), limit)
	if mapInstructorError(c, err) {
		return
	}
	out := make([]popularInstructorResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPopularInstructorResponse(row))
	}
	response.OK(c, "ok", out)
}
