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
	"mycourse-io-be/internal/taxonomy/domain" //nolint:depguard // delivery uses domain input types (UpdateTagInput, UpdateCourseLevelInput); pure mapping, no business logic
)

// Handler holds all taxonomy HTTP handlers.
type Handler struct {
	svc *application.TaxonomyService
}

// NewHandler constructs a Handler.
func NewHandler(svc *application.TaxonomyService) *Handler {
	return &Handler{svc: svc}
}

// --- Category handlers -------------------------------------------------------

func (h *Handler) listCategories(c *gin.Context) {
	var q TaxonomyBaseFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	filter := toFilter(q)
	rows, total, err := h.svc.ListCategories(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", toCategoryResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) createCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.CreateCategoryInput{
		ActorID: utils.CurrentUserID(c), Name: req.Name, Slug: req.Slug,
		Status: req.Status, ImageFileID: req.ImageFileID,
	}
	row, err := h.svc.CreateCategory(c.Request.Context(), in)
	if err != nil {
		if stderrors.Is(err, apperrors.ErrInvalidProfileMediaFile) {
			response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", toCategoryResponse(*row))
}

func (h *Handler) updateCategory(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req UpdateCategoryRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in := domain.UpdateCategoryInput{Name: req.Name, Slug: req.Slug, Status: req.Status, ImageFileID: req.ImageFileID}
	row, err := h.svc.UpdateCategory(c.Request.Context(), id, in)
	if err != nil {
		if stderrors.Is(err, apperrors.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
			return
		}
		if stderrors.Is(err, apperrors.ErrInvalidProfileMediaFile) {
			response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toCategoryResponse(*row))
}

func (h *Handler) deleteCategory(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OK(c, "deleted", nil)
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

func (h *Handler) updateTag(c *gin.Context) { //nolint:dupl // intentional parallel with updateCourseLevel; both follow the same HTTP CRUD pattern
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

func (h *Handler) updateCourseLevel(c *gin.Context) { //nolint:dupl // intentional parallel with updateTag; both follow the same HTTP CRUD pattern
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

func toCategoryResponse(c domain.Category) CategoryResponse {
	fid := ""
	if c.ImageFileID != nil {
		fid = *c.ImageFileID
	}
	return CategoryResponse{
		ID: c.ID, Name: c.Name, Slug: c.Slug, Status: c.Status,
		ImageFileID: fid, ImageURL: c.ImageFileURL,
		CreatedBy: c.CreatedBy,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

func toCategoryResponses(rows []domain.Category) []CategoryResponse {
	out := make([]CategoryResponse, len(rows))
	for i := range rows {
		out[i] = toCategoryResponse(rows[i])
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
