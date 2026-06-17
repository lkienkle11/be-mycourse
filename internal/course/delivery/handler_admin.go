package delivery

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/response"
)

func (h *Handler) listAdminCourses(c *gin.Context) {
	approval := strings.ToLower(strings.TrimSpace(c.Query("approval")))
	filter := domain.AdminCourseListFilter{ApprovedOnly: approval == "approved"}
	rows, err := h.svc.ListAdminCourses(c.Request.Context(), filter)
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

func (h *Handler) listTrashedCourses(c *gin.Context) {
	rows, err := h.svc.ListTrashedCourses(c.Request.Context())
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

func (h *Handler) trashCourse(c *gin.Context) {
	h.courseOK(c, "updated", func(courseID string) (any, error) {
		return nil, h.svc.TrashCourse(c.Request.Context(), courseID)
	})
}

func (h *Handler) restoreCourse(c *gin.Context) {
	h.courseOK(c, "updated", func(courseID string) (any, error) {
		return nil, h.svc.RestoreCourse(c.Request.Context(), courseID)
	})
}

func (h *Handler) permanentDeleteCourse(c *gin.Context) {
	withCourseID(c, func(courseID string) {
		if err := h.svc.PermanentDeleteCourse(c.Request.Context(), courseID); mapCourseError(c, err) {
			return
		}
		response.OK(c, "deleted", nil)
	})
}
