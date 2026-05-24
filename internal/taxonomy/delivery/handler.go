package delivery

import (
	"net/http"

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
	listTaxonomyItems(c, h.svc.ListTopics, toCourseTopicResponses)
}

func (h *Handler) listTopicsFull(c *gin.Context) {
	listTaxonomyItemsWithDeleted(c, h.svc.ListTopicsFull, toCourseTopicResponses, true)
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
	if err := mapTaxonomyMutationError(c, err, true); err != nil {
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
	if err := mapTaxonomyMutationError(c, err, true); err != nil {
		return
	}
	response.OK(c, "updated", toCourseTopicResponse(*row))
}

func (h *Handler) deleteTopic(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.DeleteTopic)
}

func (h *Handler) hardDeleteTopic(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.HardDeleteTopic)
}

// --- CourseOutcome handlers --------------------------------------------------

func (h *Handler) listCourseOutcomes(c *gin.Context) {
	listTaxonomyItems(c, h.svc.ListCourseOutcomes, toCourseOutcomeResponses)
}

func (h *Handler) listCourseOutcomesFull(c *gin.Context) {
	listTaxonomyItemsWithDeleted(c, h.svc.ListCourseOutcomesFull, toCourseOutcomeResponses, true)
}

func (h *Handler) createCourseOutcome(c *gin.Context) {
	createTaxonomyMutation(c, h.svc.CreateCourseOutcome, courseOutcomeInputFromCreate, toCourseOutcomeResponse, true)
}

func (h *Handler) updateCourseOutcome(c *gin.Context) {
	updateTaxonomyMutation(c, h.svc.UpdateCourseOutcome, courseOutcomeInputFromUpdate, toCourseOutcomeResponse, true)
}

func (h *Handler) deleteCourseOutcome(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.DeleteCourseOutcome)
}

func (h *Handler) hardDeleteCourseOutcome(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.HardDeleteCourseOutcome)
}

// --- CourseSkill handlers ----------------------------------------------------

func (h *Handler) listCourseSkills(c *gin.Context) {
	listTaxonomyItems(c, h.svc.ListCourseSkills, toCourseSkillResponses)
}

func (h *Handler) listCourseSkillsFull(c *gin.Context) {
	listTaxonomyItemsWithDeleted(c, h.svc.ListCourseSkillsFull, toCourseSkillResponses, true)
}

func (h *Handler) createCourseSkill(c *gin.Context) {
	createTaxonomyMutation(c, h.svc.CreateCourseSkill, courseSkillInputFromCreate, toCourseSkillResponse, false)
}

func (h *Handler) updateCourseSkill(c *gin.Context) {
	updateTaxonomyMutation(c, h.svc.UpdateCourseSkill, courseSkillInputFromUpdate, toCourseSkillResponse, false)
}

func (h *Handler) deleteCourseSkill(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.DeleteCourseSkill)
}

func (h *Handler) hardDeleteCourseSkill(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.HardDeleteCourseSkill)
}

// --- Tag handlers ------------------------------------------------------------

func (h *Handler) listTags(c *gin.Context) {
	listTaxonomyItems(c, h.svc.ListTags, toTagResponses)
}

func (h *Handler) listTagsFull(c *gin.Context) {
	listTaxonomyItemsWithDeleted(c, h.svc.ListTagsFull, toTagResponses, true)
}

func (h *Handler) createTag(c *gin.Context) {
	createTaxonomyMutation(c, h.svc.CreateTag, slugStatusInputFromCreate, slugStatusResponseFromTag, false)
}

func (h *Handler) updateTag(c *gin.Context) {
	updateSlugStatusTaxonomy[UpdateTagRequest, domain.Tag](c, h.svc.UpdateTag, slugStatusInputFromUpdate, slugStatusResponseFromTag)
}

func (h *Handler) deleteTag(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.DeleteTag)
}

func (h *Handler) hardDeleteTag(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.HardDeleteTag)
}

// --- CourseLevel handlers ----------------------------------------------------

func (h *Handler) listCourseLevels(c *gin.Context) {
	listTaxonomyItems(c, h.svc.ListCourseLevels, toCourseLevelResponses)
}

func (h *Handler) listCourseLevelsFull(c *gin.Context) {
	listTaxonomyItemsWithDeleted(c, h.svc.ListCourseLevelsFull, toCourseLevelResponses, true)
}

func (h *Handler) createCourseLevel(c *gin.Context) {
	createTaxonomyMutation(c, h.svc.CreateCourseLevel, slugStatusInputFromCreate, slugStatusResponseFromCourseLevel, false)
}

func (h *Handler) updateCourseLevel(c *gin.Context) {
	updateSlugStatusTaxonomy[UpdateCourseLevelRequest, domain.CourseLevel](c, h.svc.UpdateCourseLevel, slugStatusInputFromUpdate, slugStatusResponseFromCourseLevel)
}

func (h *Handler) deleteCourseLevel(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.DeleteCourseLevel)
}

func (h *Handler) hardDeleteCourseLevel(c *gin.Context) {
	deleteTaxonomyByID(c, h.svc.HardDeleteCourseLevel)
}

// --- mapping helpers ---------------------------------------------------------

func toFilter(q TaxonomyBaseFilter, includeDeleted bool) domain.TaxonomyFilter {
	return domain.TaxonomyFilter{
		Page: q.getPage(), PageSize: q.getPerPage(),
		Status: q.Status, Search: q.Search,
		SortBy: q.SortBy, SortDesc: q.SortDesc,
		IncludeDeleted: includeDeleted,
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
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func toCourseTopicResponses(rows []domain.CourseTopic) []CourseTopicResponse {
	return mapRowsToResponses(rows, toCourseTopicResponse)
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
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
}

func toCourseOutcomeResponses(rows []domain.CourseOutcome) []CourseOutcomeResponse {
	return mapRowsToResponses(rows, toCourseOutcomeResponse)
}

func toCourseSkillResponse(s domain.CourseSkill) CourseSkillResponse {
	child := s.Children
	if child == nil {
		child = []taxpkg.TreeNode{}
	}
	return CourseSkillResponse{
		ID: s.ID, Name: s.Name, Slug: s.Slug, Children: child, Status: s.Status, CreatedBy: s.CreatedBy,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func toCourseSkillResponses(rows []domain.CourseSkill) []CourseSkillResponse {
	return mapRowsToResponses(rows, toCourseSkillResponse)
}

func toSlugStatusResponse(t domain.Tag) SlugStatusResponse {
	return slugStatusResponse(t.ID, t.Name, t.Slug, t.Status, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
}

func toTagResponses(rows []domain.Tag) []TagResponse {
	return mapRowsToResponses(rows, toSlugStatusResponse)
}

func toCourseLevelResponses(rows []domain.CourseLevel) []CourseLevelResponse {
	return mapRowsToResponses(rows, func(cl domain.CourseLevel) SlugStatusResponse {
		return slugStatusResponse(cl.ID, cl.Name, cl.Slug, cl.Status, cl.CreatedBy, cl.CreatedAt, cl.UpdatedAt)
	})
}
