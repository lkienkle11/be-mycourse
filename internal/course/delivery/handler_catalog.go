package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

const (
	trendingCoursesDefaultLimit = 8
	trendingCoursesMaxLimit     = 24
)

func (h *Handler) listTrendingCourses(c *gin.Context) {
	limit := utils.ClampQueryLimit(c.DefaultQuery("limit", ""), trendingCoursesDefaultLimit, trendingCoursesMaxLimit)
	rows, err := h.svc.ListTrendingCourses(c.Request.Context(), limit)
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}
