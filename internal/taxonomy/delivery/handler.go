package delivery

import (
	stderrors "errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
	"mycourse-io-be/internal/taxonomy/application"
	"mycourse-io-be/internal/taxonomy/domain"
	taxpkg "mycourse-io-be/pkg/taxonomy"
)

// Handler holds all taxonomy HTTP handlers.
type Handler struct {
	svc *application.TaxonomyService
}

// NewHandler constructs a Handler.
func NewHandler(svc *application.TaxonomyService) *Handler {
	return &Handler{svc: svc}
}

// --- CourseTopic handlers ----------------------------------------------------

func (h *Handler) listTopics(c *gin.Context) {
	var q TaxonomyBaseFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := h.svc.ListTopics(c.Request.Context(), toFilter(q))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", toCourseTopicResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) createTopic(c *gin.Context) {
	var req CreateCourseTopicRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.CreateCourseTopicInput{
		ActorID: utils.CurrentUserID(c), Name: req.Name, Slug: req.Slug,
		Status: req.Status, ImageFileID: req.ImageFileID, ChildTopics: req.ChildTopics,
	}
	row, err := h.svc.CreateTopic(c.Request.Context(), in)
	if err := h.mapTopicMutationError(c, err); err != nil {
		return
	}
	response.Created(c, "created", toCourseTopicResponse(*row))
}

func (h *Handler) updateTopic(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req UpdateCourseTopicRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.UpdateCourseTopicInput{
		Name: req.Name, Slug: req.Slug, Status: req.Status,
		ImageFileID: req.ImageFileID, ChildTopics: req.ChildTopics,
	}
	row, err := h.svc.UpdateTopic(c.Request.Context(), id, in)
	if err := h.mapTopicMutationError(c, err); err != nil {
		return
	}
	response.OK(c, "updated", toCourseTopicResponse(*row))
}

func (h *Handler) deleteTopic(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := h.svc.DeleteTopic(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func (h *Handler) mapTopicMutationError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, apperrors.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
		return err
	}
	if stderrors.Is(err, apperrors.ErrInvalidProfileMediaFile) {
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

// --- CourseOutcome handlers --------------------------------------------------

func (h *Handler) listCourseOutcomes(c *gin.Context) {
	var q TaxonomyBaseFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := h.svc.ListCourseOutcomes(c.Request.Context(), toFilter(q))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", toCourseOutcomeResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) createCourseOutcome(c *gin.Context) {
	var req CreateCourseOutcomeRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.CreateCourseOutcomeInput{
		ActorID: utils.CurrentUserID(c), ShortDescription: req.ShortDescription,
		Description: req.Description, Status: req.Status, ImageFileID: req.ImageFileID,
	}
	row, err := h.svc.CreateCourseOutcome(c.Request.Context(), in)
	if err := h.mapOutcomeMutationError(c, err); err != nil {
		return
	}
	response.Created(c, "created", toCourseOutcomeResponse(*row))
}

func (h *Handler) updateCourseOutcome(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req UpdateCourseOutcomeRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.UpdateCourseOutcomeInput{
		ShortDescription: req.ShortDescription, Description: req.Description,
		Status: req.Status, ImageFileID: req.ImageFileID,
	}
	row, err := h.svc.UpdateCourseOutcome(c.Request.Context(), id, in)
	if err := h.mapOutcomeMutationError(c, err); err != nil {
		return
	}
	response.OK(c, "updated", toCourseOutcomeResponse(*row))
}

func (h *Handler) deleteCourseOutcome(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := h.svc.DeleteCourseOutcome(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func (h *Handler) mapOutcomeMutationError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, apperrors.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
		return err
	}
	if stderrors.Is(err, apperrors.ErrInvalidProfileMediaFile) {
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

// --- CourseSkill handlers ----------------------------------------------------

func (h *Handler) listCourseSkills(c *gin.Context) {
	var q TaxonomyBaseFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := h.svc.ListCourseSkills(c.Request.Context(), toFilter(q))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", toCourseSkillResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) createCourseSkill(c *gin.Context) {
	var req CreateCourseSkillRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.CreateCourseSkillInput{
		ActorID: utils.CurrentUserID(c), Name: req.Name, Slug: req.Slug,
		Status: req.Status, Children: req.Children,
	}
	row, err := h.svc.CreateCourseSkill(c.Request.Context(), in)
	if err := h.mapSkillMutationError(c, err); err != nil {
		return
	}
	response.Created(c, "created", toCourseSkillResponse(*row))
}

func (h *Handler) updateCourseSkill(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req UpdateCourseSkillRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.UpdateCourseSkillInput{Name: req.Name, Slug: req.Slug, Status: req.Status, Children: req.Children}
	row, err := h.svc.UpdateCourseSkill(c.Request.Context(), id, in)
	if err := h.mapSkillMutationError(c, err); err != nil {
		return
	}
	response.OK(c, "updated", toCourseSkillResponse(*row))
}

func (h *Handler) deleteCourseSkill(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := h.svc.DeleteCourseSkill(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func (h *Handler) mapSkillMutationError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, apperrors.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
		return err
	}
	if stderrors.Is(err, application.ErrTaxonomyValidation) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return err
	}
	response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
	return err
}

// --- Tag handlers ------------------------------------------------------------

func (h *Handler) listTags(c *gin.Context) {
	var q TaxonomyBaseFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := h.svc.ListTags(c.Request.Context(), toFilter(q))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", toTagResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) createTag(c *gin.Context) {
	var req CreateTagRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.CreateTagInput{ActorID: utils.CurrentUserID(c), Name: req.Name, Slug: req.Slug, Status: req.Status}
	row, err := h.svc.CreateTag(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", toTagResponse(*row))
}

func (h *Handler) updateTag(c *gin.Context) { //nolint:dupl // intentional parallel with updateCourseLevel
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req UpdateTagRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.UpdateTag(c.Request.Context(), id, domain.UpdateTagInput{Name: req.Name, Slug: req.Slug, Status: req.Status})
	if err != nil {
		if stderrors.Is(err, apperrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toTagResponse(*row))
}

func (h *Handler) deleteTag(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := h.svc.DeleteTag(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

// --- CourseLevel handlers ----------------------------------------------------

func (h *Handler) listCourseLevels(c *gin.Context) {
	var q TaxonomyBaseFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := h.svc.ListCourseLevels(c.Request.Context(), toFilter(q))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", toCourseLevelResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) createCourseLevel(c *gin.Context) {
	var req CreateCourseLevelRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.CreateCourseLevelInput{ActorID: utils.CurrentUserID(c), Name: req.Name, Slug: req.Slug, Status: req.Status}
	row, err := h.svc.CreateCourseLevel(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", toCourseLevelResponse(*row))
}

func (h *Handler) updateCourseLevel(c *gin.Context) { //nolint:dupl // intentional parallel with updateTag
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req UpdateCourseLevelRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	row, err := h.svc.UpdateCourseLevel(c.Request.Context(), id, domain.UpdateCourseLevelInput{Name: req.Name, Slug: req.Slug, Status: req.Status})
	if err != nil {
		if stderrors.Is(err, apperrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toCourseLevelResponse(*row))
}

func (h *Handler) deleteCourseLevel(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := h.svc.DeleteCourseLevel(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

// --- mapping helpers ---------------------------------------------------------

func toFilter(q TaxonomyBaseFilter) domain.TaxonomyFilter {
	return domain.TaxonomyFilter{
		Page: q.getPage(), PageSize: q.getPerPage(),
		Status: q.Status, Search: q.Search,
		SortBy: q.SortBy, SortDesc: q.SortDesc,
	}
}

func toCourseTopicResponse(t domain.CourseTopic) CourseTopicResponse {
	fid := ""
	if t.ImageFileID != nil {
		fid = *t.ImageFileID
	}
	child := t.ChildTopics
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return CourseTopicResponse{
		ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
		ImageFileID: fid, ImageURL: t.ImageFileURL, ChildTopics: child,
		CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

func toCourseTopicResponses(rows []domain.CourseTopic) []CourseTopicResponse {
	out := make([]CourseTopicResponse, len(rows))
	for i := range rows {
		out[i] = toCourseTopicResponse(rows[i])
	}
	return out
}

func toCourseOutcomeResponse(o domain.CourseOutcome) CourseOutcomeResponse {
	fid := ""
	if o.ImageFileID != nil {
		fid = *o.ImageFileID
	}
	desc := o.Description
	if desc == nil {
		desc = []string{}
	}
	return CourseOutcomeResponse{
		ID: o.ID, ShortDescription: o.ShortDescription, Description: desc,
		ImageFileID: fid, ImageURL: o.ImageFileURL, Status: o.Status, CreatedBy: o.CreatedBy,
		CreatedAt: o.CreatedAt.Format(time.RFC3339), UpdatedAt: o.UpdatedAt.Format(time.RFC3339),
	}
}

func toCourseOutcomeResponses(rows []domain.CourseOutcome) []CourseOutcomeResponse {
	out := make([]CourseOutcomeResponse, len(rows))
	for i := range rows {
		out[i] = toCourseOutcomeResponse(rows[i])
	}
	return out
}

func toCourseSkillResponse(s domain.CourseSkill) CourseSkillResponse {
	child := s.Children
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return CourseSkillResponse{
		ID: s.ID, Name: s.Name, Slug: s.Slug, Children: child, Status: s.Status, CreatedBy: s.CreatedBy,
		CreatedAt: s.CreatedAt.Format(time.RFC3339), UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
}

func toCourseSkillResponses(rows []domain.CourseSkill) []CourseSkillResponse {
	out := make([]CourseSkillResponse, len(rows))
	for i := range rows {
		out[i] = toCourseSkillResponse(rows[i])
	}
	return out
}

func toTagResponse(t domain.Tag) TagResponse {
	return TagResponse{
		ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status, CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt.Format(time.RFC3339), UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

func toTagResponses(rows []domain.Tag) []TagResponse {
	out := make([]TagResponse, len(rows))
	for i := range rows {
		out[i] = toTagResponse(rows[i])
	}
	return out
}

func toCourseLevelResponse(cl domain.CourseLevel) CourseLevelResponse {
	return CourseLevelResponse{
		ID: cl.ID, Name: cl.Name, Slug: cl.Slug, Status: cl.Status, CreatedBy: cl.CreatedBy,
		CreatedAt: cl.CreatedAt.Format(time.RFC3339), UpdatedAt: cl.UpdatedAt.Format(time.RFC3339),
	}
}

func toCourseLevelResponses(rows []domain.CourseLevel) []CourseLevelResponse {
	out := make([]CourseLevelResponse, len(rows))
	for i := range rows {
		out[i] = toCourseLevelResponse(rows[i])
	}
	return out
}
