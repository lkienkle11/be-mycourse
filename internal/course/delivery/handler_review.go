package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func (h *Handler) submitForReview(c *gin.Context) {
	h.courseOK(c, "updated", func(courseID uint) (any, error) {
		return h.svc.SubmitForReview(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) reopenDraft(c *gin.Context) {
	h.courseOK(c, "updated", func(courseID uint) (any, error) {
		return h.svc.ReopenDraft(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) listPendingReviews(c *gin.Context) {
	rows, err := h.svc.ListPendingReviews(c.Request.Context())
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

func (h *Handler) approveDraft(c *gin.Context) {
	h.courseOK(c, "updated", func(courseID uint) (any, error) {
		return h.svc.ApproveDraft(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) rejectDraft(c *gin.Context) {
	courseBodyOK(h, c, "updated", func(courseID uint, req *rejectDraftRequest) (any, error) {
		return h.svc.RejectDraft(c.Request.Context(), courseID, utils.CurrentUserID(c), req.Reason)
	})
}
