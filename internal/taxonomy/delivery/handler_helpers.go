package delivery

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/httpx"
	"mycourse-io-be/internal/shared/response"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
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

func getTaxonomyByID[TRow any, TResp any](
	c *gin.Context,
	getFn func(context.Context, string, string, bool) (*TRow, error),
	toResponse func(TRow) TResp,
) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var q TaxonomyGetQuery
	_ = c.ShouldBindQuery(&q)
	viewEdit := strings.EqualFold(strings.TrimSpace(q.View), "edit")
	row, err := getFn(c.Request.Context(), id, q.Locale, viewEdit)
	if err != nil {
		if stderrors.Is(err, apperrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "ok", toResponse(*row))
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
	if stderrors.Is(err, domain.ErrTaxonomyOptimisticLock) {
		response.Fail(c, http.StatusConflict, apperrors.Conflict, err.Error(), nil)
		return err
	}
	if stderrors.Is(err, domain.ErrTaxonomyCanonicalConflict) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
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
	updateTaxonomyMutation(c, updateFn, toInput, toResponse, false)
}

func outcomeTranslationsFromDTO(in map[string]OutcomeTranslationDTO) map[string]domain.OutcomeTranslation {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]domain.OutcomeTranslation, len(in))
	for k, v := range in {
		out[k] = domain.OutcomeTranslation{ShortDescription: v.ShortDescription, Description: v.Description}
	}
	return out
}

func outcomeTranslationsToDTO(in map[string]domain.OutcomeTranslation) map[string]OutcomeTranslationDTO {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]OutcomeTranslationDTO, len(in))
	for k, v := range in {
		desc := v.Description
		if desc == nil {
			desc = []string{}
		}
		out[k] = OutcomeTranslationDTO{ShortDescription: v.ShortDescription, Description: desc}
	}
	return out
}

func courseOutcomeInputFromCreate(req CreateCourseOutcomeRequest, actorID string) domain.CreateCourseOutcomeInput {
	return domain.CreateCourseOutcomeInput{
		ActorID: actorID, ShortDescription: req.ShortDescription,
		Description: req.Description, Status: req.Status, ImageFileID: req.ImageFileID,
		Translations: outcomeTranslationsFromDTO(req.Translations),
	}
}

func courseOutcomeInputFromUpdate(req UpdateCourseOutcomeRequest) domain.UpdateCourseOutcomeInput {
	return domain.UpdateCourseOutcomeInput{
		ShortDescription: req.ShortDescription, Description: req.Description,
		Status: req.Status, ImageFileID: req.ImageFileID,
		Translations:       outcomeTranslationsFromDTO(req.Translations),
		ExpectedRowVersion: req.ExpectedRowVersion,
	}
}

func courseSkillInputFromCreate(req CreateCourseSkillRequest, actorID string) domain.CreateCourseSkillInput {
	return domain.CreateCourseSkillInput{
		ActorID: actorID, Name: req.Name, Status: req.Status, Children: req.Children,
		Translations: req.Translations,
	}
}

func courseSkillInputFromUpdate(req UpdateCourseSkillRequest) domain.UpdateCourseSkillInput {
	return domain.UpdateCourseSkillInput{
		Name: req.Name, Status: req.Status, Children: req.Children,
		Translations: req.Translations, ExpectedRowVersion: req.ExpectedRowVersion,
	}
}

func slugStatusInputFromCreate(req CreateTagRequest, actorID string) domain.CreateTagInput {
	return domain.CreateTagInput{ActorID: actorID, Name: req.Name, Status: req.Status, Translations: req.Translations}
}

func slugStatusInputFromUpdate(req UpdateTagRequest) domain.UpdateTagInput {
	return domain.UpdateTagInput{
		Name: req.Name, Status: req.Status, Translations: req.Translations,
		ExpectedRowVersion: req.ExpectedRowVersion,
	}
}

func slugStatusResponseFromTag(t domain.Tag) SlugStatusResponse {
	f := slugStatusFields{ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status}
	f.ResolvedLocale = t.ResolvedLocale
	f.AvailableLocales = t.AvailableLocales
	f.Translations = t.Translations
	f.RowVersion = t.RowVersion
	f.CreatedBy = t.CreatedBy
	f.CreatedAt = t.CreatedAt
	f.UpdatedAt = t.UpdatedAt
	return slugStatusResponse(f)
}

func slugStatusResponseFromCourseLevel(cl domain.CourseLevel) SlugStatusResponse {
	return slugStatusResponse(slugStatusFields{
		ID: cl.ID, Name: cl.Name, Slug: cl.Slug, Status: cl.Status,
		ResolvedLocale: cl.ResolvedLocale, AvailableLocales: cl.AvailableLocales,
		Translations: cl.Translations, RowVersion: cl.RowVersion,
		CreatedBy: cl.CreatedBy, CreatedAt: cl.CreatedAt, UpdatedAt: cl.UpdatedAt,
	})
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

type slugStatusFields struct {
	ID               string
	Name             string
	Slug             string
	Status           string
	ResolvedLocale   string
	AvailableLocales []string
	Translations     map[string]taxpkg.NodeTranslation
	RowVersion       int64
	CreatedBy        *string
	CreatedAt        int64
	UpdatedAt        int64
}

func slugStatusResponse(f slugStatusFields) SlugStatusResponse {
	return SlugStatusResponse{
		ID: f.ID, Name: f.Name, Slug: f.Slug, Status: f.Status, ResolvedLocale: f.ResolvedLocale,
		AvailableLocales: f.AvailableLocales, Translations: f.Translations,
		RowVersion: editRowVersion(f.Translations != nil || len(f.AvailableLocales) > 0, f.RowVersion),
		CreatedBy:  f.CreatedBy, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
	}
}

func editRowVersion(isEdit bool, rowVersion int64) int64 {
	if !isEdit {
		return 0
	}
	return rowVersion
}

func mapRowsToResponses[D any, R any](rows []D, mapOne func(D) R) []R {
	out := make([]R, len(rows))
	for i := range rows {
		out[i] = mapOne(rows[i])
	}
	return out
}
