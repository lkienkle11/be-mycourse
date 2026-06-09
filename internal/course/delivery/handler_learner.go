package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func (h *Handler) listPublishedCourses(c *gin.Context) {
	rows, err := h.svc.ListPublishedCourses(c.Request.Context())
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

func (h *Handler) getLearningCourse(c *gin.Context) {
	h.courseOK(c, "ok", func(courseID string) (any, error) {
		return h.svc.GetLearningCourse(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) enroll(c *gin.Context) {
	h.courseCreated(c, "created", func(courseID string) (any, error) {
		return h.svc.Enroll(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) getProgress(c *gin.Context) {
	h.courseOK(c, "ok", func(courseID string) (any, error) {
		return h.svc.GetProgress(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) saveProgress(c *gin.Context) {
	courseBodyOK(h, c, "updated", func(courseID string, req *saveProgressRequest) (any, error) {
		return h.svc.SaveProgress(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.SaveProgressInput{
			StableContentID: req.StableContentID,
			ContentType:     req.ContentType,
			Status:          req.Status,
			Score:           req.Score,
			QuizAttempt:     req.QuizAttempt,
		})
	})
}
