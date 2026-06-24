package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func (h *Handler) submitForReview(c *gin.Context) {
	h.courseOK(c, "updated", func(courseID string) (any, error) {
		return h.svc.SubmitForReview(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) reopenDraft(c *gin.Context) {
	h.courseOK(c, "updated", func(courseID string) (any, error) {
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
	courseBodyUpdated(h, c, func(courseID string, req *approveDraftRequest) (any, error) {
		return h.svc.ApproveDraft(c.Request.Context(), courseID, utils.CurrentUserID(c), req.ApprovalNote)
	})
}

func (h *Handler) rejectDraft(c *gin.Context) {
	courseBodyUpdated(h, c, func(courseID string, req *rejectDraftRequest) (any, error) {
		return h.svc.RejectDraft(c.Request.Context(), courseID, utils.CurrentUserID(c), req.Reason)
	})
}

func (h *Handler) listReviewHistory(c *gin.Context) {
	withCourseID(c, func(courseID string) {
		var q listReviewHistoryQuery
		_ = c.ShouldBindQuery(&q)
		rows, total, err := h.svc.ListReviewHistory(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.ReviewHistoryFilter{
			Page:    q.GetPage(),
			PerPage: q.GetPerPage(),
			Status:  q.Status,
		})
		if mapCourseError(c, err) {
			return
		}
		response.OKPaginated(c, "ok", rows, utils.BuildPage(q.GetPage(), q.GetPerPage(), total))
	})
}
