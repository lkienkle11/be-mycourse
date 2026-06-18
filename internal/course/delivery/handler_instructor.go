package delivery

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func (h *Handler) listEditableCourses(c *gin.Context) {
	rows, err := h.svc.ListEditableCourses(c.Request.Context(), utils.CurrentUserID(c))
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

func (h *Handler) createCourse(c *gin.Context) {
	withBody(c, func(req *createCourseRequest) {
		row, err := h.svc.CreateCourse(c.Request.Context(), domain.CreateCourseInput{
			ActorUserID: utils.CurrentUserID(c),
			Title:       req.Title,
		})
		if mapCourseError(c, err) {
			return
		}
		response.Created(c, "created", row)
	})
}

func (h *Handler) getCourseDetail(c *gin.Context) {
	includeOutline := c.DefaultQuery("include_outline", "true") != "false"
	h.courseOK(c, "ok", func(courseID string) (any, error) {
		return h.svc.GetCourseDetail(c.Request.Context(), courseID, utils.CurrentUserID(c), true, includeOutline)
	})
}

func (h *Handler) prepareDraft(c *gin.Context) {
	h.courseOK(c, "ok", func(courseID string) (any, error) {
		return h.svc.PrepareDraft(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) updateBasicInfo(c *gin.Context) {
	courseBodyOK(h, c, "updated", func(courseID string, req *updateBasicInfoRequest) (any, error) {
		title := strings.TrimSpace(req.Title)
		shortDescription := strings.TrimSpace(req.ShortDescription)
		aboutCourse := strings.TrimSpace(req.AboutCourse)
		thumbnailFileID := strings.TrimSpace(req.ThumbnailFileID)
		courseLevelID := strings.TrimSpace(req.CourseLevelID)
		courseTopicID := strings.TrimSpace(req.CourseTopicID)
		return h.svc.UpdateBasicInfo(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.UpdateBasicInfoInput{
			ActorUserID:        utils.CurrentUserID(c),
			ExpectedRowVersion: req.ExpectedRowVersion,
			Title:              &title,
			ShortDescription:   &shortDescription,
			AboutCourse:        &aboutCourse,
			ThumbnailFileID:    &thumbnailFileID,
			PreviewVideoFileID: optionalTrimmedStringPtr(req.PreviewVideoFileID),
			CourseLevelID:      &courseLevelID,
			CourseTopicID:      &courseTopicID,
			TagIDs:             req.TagIDs,
			SkillIDs:           req.SkillIDs,
			OutcomeIDs:         req.OutcomeIDs,
		})
	})
}

func optionalTrimmedStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (h *Handler) deleteCourse(c *gin.Context) {
	withCourseID(c, func(courseID string) {
		if err := h.svc.DeleteCourse(c.Request.Context(), courseID, utils.CurrentUserID(c)); mapCourseError(c, err) {
			return
		}
		response.OK(c, "deleted", gin.H{"course_id": courseID})
	})
}

func (h *Handler) listCollaborators(c *gin.Context) {
	h.courseOK(c, "ok", func(courseID string) (any, error) {
		return h.svc.ListCollaborators(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

func (h *Handler) addCollaborator(c *gin.Context) {
	courseBodyOK(h, c, "updated", func(courseID string, req *addCollaboratorRequest) (any, error) {
		return h.svc.AddCollaborator(c.Request.Context(), courseID, utils.CurrentUserID(c), req.UserID, req.Role)
	})
}

func (h *Handler) removeCollaborator(c *gin.Context) {
	h.courseUUIDParamOK(c, "userId", "invalid user id", "updated", func(courseID string, userID string) (any, error) {
		return h.svc.RemoveCollaborator(c.Request.Context(), courseID, utils.CurrentUserID(c), userID)
	})
}
