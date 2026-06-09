package delivery

import (
	"context"
	stderrors "errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/httpx"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
	"mycourse-io-be/internal/taxonomy/application"
	"mycourse-io-be/internal/taxonomy/domain"
)

func listTaxonomyItems[TRow any, TResp any](
	c *gin.Context,
	listFn func(context.Context, domain.TaxonomyFilter) ([]TRow, int64, error),
	toResponses func([]TRow) []TResp,
) {
	listTaxonomyItemsWithDeleted(c, listFn, toResponses, false)
}

func listTaxonomyItemsWithDeleted[TRow any, TResp any](
	c *gin.Context,
	listFn func(context.Context, domain.TaxonomyFilter) ([]TRow, int64, error),
	toResponses func([]TRow) []TResp,
	includeDeleted bool,
) {
	httpx.ListPaginated(c,
		func(q *TaxonomyBaseFilter) error { return c.ShouldBindQuery(q) },
		func(ctx context.Context, q TaxonomyBaseFilter) ([]TRow, int64, error) {
			return listFn(ctx, toFilter(q, includeDeleted))
		},
		func(q TaxonomyBaseFilter) (int, int) { return q.getPage(), q.getPerPage() },
		toResponses,
	)
}

func deleteTaxonomyByID(c *gin.Context, deleteFn func(context.Context, string) error) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := deleteFn(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func mapTaxonomyMutationError(c *gin.Context, err error, withProfileMedia bool) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, apperrors.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
		return err
	}
	if withProfileMedia && stderrors.Is(err, apperrors.ErrInvalidProfileMediaFile) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return err
	}
	if stderrors.Is(err, application.ErrTaxonomyValidation) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return err
	}
	response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
	return err
}

func updateSlugStatusTaxonomy[Req any, Row any](
	c *gin.Context,
	updateFn func(context.Context, string, domain.UpdateTagInput) (*Row, error),
	toInput func(Req) domain.UpdateTagInput,
	toResponse func(Row) SlugStatusResponse,
) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := updateFn(c.Request.Context(), id, toInput(req))
	if err != nil {
		if stderrors.Is(err, apperrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toResponse(*row))
}

func courseOutcomeInputFromCreate(req CreateCourseOutcomeRequest, actorID string) domain.CreateCourseOutcomeInput {
	return domain.CreateCourseOutcomeInput{
		ActorID: actorID, ShortDescription: req.ShortDescription,
		Description: req.Description, Status: req.Status, ImageFileID: req.ImageFileID,
	}
}

func courseOutcomeInputFromUpdate(req UpdateCourseOutcomeRequest) domain.UpdateCourseOutcomeInput {
	return domain.UpdateCourseOutcomeInput{
		ShortDescription: req.ShortDescription, Description: req.Description,
		Status: req.Status, ImageFileID: req.ImageFileID,
	}
}

func courseSkillInputFromCreate(req CreateCourseSkillRequest, actorID string) domain.CreateCourseSkillInput {
	return domain.CreateCourseSkillInput{
		ActorID: actorID, Name: req.Name, Status: req.Status, Children: req.Children,
	}
}

func courseSkillInputFromUpdate(req UpdateCourseSkillRequest) domain.UpdateCourseSkillInput {
	return domain.UpdateCourseSkillInput{Name: req.Name, Status: req.Status, Children: req.Children}
}

func slugStatusInputFromCreate(req CreateTagRequest, actorID string) domain.CreateTagInput {
	return domain.CreateTagInput{ActorID: actorID, Name: req.Name, Status: req.Status}
}

func slugStatusInputFromUpdate(req UpdateTagRequest) domain.UpdateTagInput {
	return domain.UpdateTagInput{Name: req.Name, Status: req.Status}
}

func slugStatusResponseFromTag(t domain.Tag) SlugStatusResponse {
	return slugStatusResponse(t.ID, t.Name, t.Slug, t.Status, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
}

func slugStatusResponseFromCourseLevel(cl domain.CourseLevel) SlugStatusResponse {
	return slugStatusResponse(cl.ID, cl.Name, cl.Slug, cl.Status, cl.CreatedBy, cl.CreatedAt, cl.UpdatedAt)
}

func updateTaxonomyMutation[Req any, In any, Row any, Resp any](
	c *gin.Context,
	updateFn func(context.Context, string, In) (*Row, error),
	toInput func(Req) In,
	toResponse func(Row) Resp,
	withProfileMedia bool,
) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req Req
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := updateFn(c.Request.Context(), id, toInput(req))
	if err := mapTaxonomyMutationError(c, err, withProfileMedia); err != nil {
		return
	}
	response.OK(c, "updated", toResponse(*row))
}

func createTaxonomyMutation[Req any, In any, Row any, Resp any](
	c *gin.Context,
	createFn func(context.Context, In) (*Row, error),
	toInput func(Req, string) In,
	toResponse func(Row) Resp,
	withProfileMedia bool,
) {
	var req Req
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := toInput(req, utils.CurrentUserID(c))
	row, err := createFn(c.Request.Context(), in)
	if err := mapTaxonomyMutationError(c, err, withProfileMedia); err != nil {
		return
	}
	response.Created(c, "created", toResponse(*row))
}

func slugStatusResponse(id string, name, slug, status string, createdBy *string, createdAt, updatedAt int64) SlugStatusResponse {
	return SlugStatusResponse{
		ID: id, Name: name, Slug: slug, Status: status, CreatedBy: createdBy,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func mapRowsToResponses[D any, R any](rows []D, mapOne func(D) R) []R {
	out := make([]R, len(rows))
	for i := range rows {
		out[i] = mapOne(rows[i])
	}
	return out
}
