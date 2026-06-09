package delivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/httpx"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
)

func (h *Handler) listProfiles(c *gin.Context) {
	httpx.ListPaginated(c,
		func(q *listQuery) error { return c.ShouldBindQuery(q) },
		func(ctx context.Context, q listQuery) ([]domain.Profile, int64, error) {
			return h.svc.ListProfiles(ctx, domain.ProfileFilter{Page: q.getPage(), PageSize: q.getPerPage()})
		},
		func(q listQuery) (int, int) { return q.getPage(), q.getPerPage() },
		func(rows []domain.Profile) []applicationResponse {
			out := make([]applicationResponse, len(rows))
			for i, r := range rows {
				out[i] = profileToResponse(r)
			}
			return out
		},
	)
}

func (h *Handler) getProfileMe(c *gin.Context) { h.respondProfileMe(c) }

func (h *Handler) getProfileByUser(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		failInvalidID(c)
		return
	}
	row, err := h.svc.GetProfileByUserID(c.Request.Context(), userID)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", profileToResponse(*row))
}

func (h *Handler) upsertProfile(c *gin.Context) {
	var req profileBody
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	userID, ok := parseUserIDParam(c)
	if !ok {
		userID = utils.CurrentUserID(c)
	}
	row, err := h.svc.UpsertProfile(c.Request.Context(), domain.UpsertProfileInput{
		UserID: userID, ProfilePayload: req.toPayload(),
	})
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", gin.H{"id": row.ID, "user_id": row.UserID})
}

func (h *Handler) deleteProfile(c *gin.Context) {
	id, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteProfile(c.Request.Context(), id); mapInstructorError(c, err) {
		return
	}
	response.OK(c, "deleted", nil)
}
