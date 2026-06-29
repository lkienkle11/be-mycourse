package delivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/instructor/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/validate"
)

func (h *Handler) listRoster(c *gin.Context) {
	listPaginatedWithQuery(c, h.fetchRosterMembers, mapRosterResponses)
}

func (h *Handler) listRosterCandidates(c *gin.Context) {
	listPaginatedWithQuery(c, h.fetchRosterCandidates, mapRosterCandidateResponses)
}

func (h *Handler) fetchRosterMembers(ctx context.Context, q listQuery) ([]domain.RosterMember, int64, error) {
	return h.svc.ListRoster(ctx, domain.RosterFilter{
		Page: q.getPage(), PageSize: q.getRosterPerPage(), Search: q.Search,
	})
}

func (h *Handler) fetchRosterCandidates(ctx context.Context, q listQuery) ([]domain.RosterCandidate, int64, error) {
	return h.svc.ListRosterCandidates(ctx, domain.RosterCandidateFilter{
		Page: q.getPage(), PageSize: q.getRosterPerPage(), Search: q.Search,
	})
}

func mapRosterResponses(rows []domain.RosterMember) []rosterResponse {
	out := make([]rosterResponse, len(rows))
	for i, row := range rows {
		out[i] = toRosterResponse(row)
	}
	return out
}

func mapRosterCandidateResponses(rows []domain.RosterCandidate) []rosterCandidateResponse {
	out := make([]rosterCandidateResponse, len(rows))
	for i, row := range rows {
		out[i] = toRosterCandidateResponse(row)
	}
	return out
}

func (h *Handler) addRosterBulk(c *gin.Context) {
	var req addRosterBulkRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	result, err := h.svc.AddRosterBulk(c.Request.Context(), req.UserIDs)
	if mapInstructorError(c, err) {
		return
	}
	response.OK(c, "ok", toRosterBulkResponse(result))
}
