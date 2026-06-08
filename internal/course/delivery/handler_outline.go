package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func (h *Handler) createSection(c *gin.Context) {
	courseBodyCreated(h, c, "created", func(courseID uint, req *sectionRequest) (any, error) {
		return h.svc.CreateSection(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.UpsertSectionInput{
			Title:       req.Title,
			Description: req.Description,
		})
	})
}

func (h *Handler) updateSection(c *gin.Context) {
	courseParamBodyOK(h, c, "sectionId", "invalid section id", "updated", func(courseID, sectionID uint, req *sectionRequest) (any, error) {
		return h.svc.UpdateSection(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.UpsertSectionInput{
			SectionID:          &sectionID,
			ExpectedRowVersion: req.ExpectedRowVersion,
			Title:              req.Title,
			Description:        req.Description,
		})
	})
}

func (h *Handler) deleteSection(c *gin.Context) {
	h.courseParamOK(c, "sectionId", "invalid section id", "deleted", func(courseID, sectionID uint) (any, error) {
		return h.svc.DeleteSection(c.Request.Context(), courseID, utils.CurrentUserID(c), sectionID)
	})
}

func (h *Handler) reorderSections(c *gin.Context) {
	courseBodyOK(h, c, "updated", func(courseID uint, req *reorderRequest) (any, error) {
		return h.svc.ReorderSections(c.Request.Context(), courseID, utils.CurrentUserID(c), req.OrderedStableIDs)
	})
}

func (h *Handler) createLesson(c *gin.Context) { h.upsertLesson(c, true) }
func (h *Handler) updateLesson(c *gin.Context) { h.upsertLesson(c, false) }

func (h *Handler) upsertLesson(c *gin.Context, create bool) {
	handler := func(courseID uint, lessonID *uint, req *lessonRequest) {
		input := domain.UpsertLessonInput{
			LessonID:           lessonID,
			SectionID:          req.SectionID,
			ExpectedRowVersion: req.ExpectedRowVersion,
			Title:              req.Title,
			Summary:            req.Summary,
		}
		if create {
			row, err := h.svc.CreateLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), input)
			if mapCourseError(c, err) {
				return
			}
			response.Created(c, "created", row)
			return
		}
		row, err := h.svc.UpdateLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), input)
		if mapCourseError(c, err) {
			return
		}
		response.OK(c, "updated", row)
	}

	if create {
		withCourseAndBody[lessonRequest](c, func(courseID uint, req *lessonRequest) {
			handler(courseID, nil, req)
		})
		return
	}

	withCourseParamAndBody[lessonRequest](c, "lessonId", "invalid lesson id", func(courseID, lessonID uint, req *lessonRequest) {
		handler(courseID, &lessonID, req)
	})
}

func (h *Handler) deleteLesson(c *gin.Context) {
	h.courseParamOK(c, "lessonId", "invalid lesson id", "deleted", func(courseID, lessonID uint) (any, error) {
		return h.svc.DeleteLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), lessonID)
	})
}

func (h *Handler) reorderLessons(c *gin.Context) {
	h.reorderOutlineChildren(c, "sectionId", "invalid section id", func(courseID, actorUserID, sectionID uint, orderedStableIDs []string) (any, error) {
		return h.svc.ReorderLessons(c.Request.Context(), courseID, actorUserID, sectionID, orderedStableIDs)
	})
}

func (h *Handler) createSubLesson(c *gin.Context) { h.upsertSubLesson(c, true) }
func (h *Handler) updateSubLesson(c *gin.Context) { h.upsertSubLesson(c, false) }

func (h *Handler) upsertSubLesson(c *gin.Context, create bool) {
	handler := func(courseID uint, subLessonID *uint, req *subLessonRequest) {
		input := domain.UpsertSubLessonInput{
			SubLessonID:        subLessonID,
			LessonID:           req.LessonID,
			ExpectedRowVersion: req.ExpectedRowVersion,
			Title:              req.Title,
			Kind:               req.Kind,
			IsPreview:          req.IsPreview,
			Video:              toVideoContent(req.Video),
			Text:               toTextContent(req.Text),
			Quiz:               toQuizContent(req.Quiz),
		}
		if create {
			row, err := h.svc.CreateSubLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), input)
			if mapCourseError(c, err) {
				return
			}
			response.Created(c, "created", row)
			return
		}
		row, err := h.svc.UpdateSubLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), input)
		if mapCourseError(c, err) {
			return
		}
		response.OK(c, "updated", row)
	}

	if create {
		withCourseAndBody[subLessonRequest](c, func(courseID uint, req *subLessonRequest) {
			handler(courseID, nil, req)
		})
		return
	}

	withCourseParamAndBody[subLessonRequest](c, "subLessonId", "invalid sub-lesson id", func(courseID, subLessonID uint, req *subLessonRequest) {
		handler(courseID, &subLessonID, req)
	})
}

func (h *Handler) deleteSubLesson(c *gin.Context) {
	h.courseParamOK(c, "subLessonId", "invalid sub-lesson id", "deleted", func(courseID, subLessonID uint) (any, error) {
		return h.svc.DeleteSubLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), subLessonID)
	})
}

func (h *Handler) reorderSubLessons(c *gin.Context) {
	h.reorderOutlineChildren(c, "lessonId", "invalid lesson id", func(courseID, actorUserID, lessonID uint, orderedStableIDs []string) (any, error) {
		return h.svc.ReorderSubLessons(c.Request.Context(), courseID, actorUserID, lessonID, orderedStableIDs)
	})
}

func (h *Handler) acquireLease(c *gin.Context) {
	courseBodyOK(h, c, "ok", func(courseID uint, req *leaseAcquireRequest) (any, error) {
		return h.svc.AcquireLease(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.AcquireLeaseInput{
			CourseVersionID:  req.CourseVersionID,
			ResourceType:     req.ResourceType,
			ResourceStableID: req.ResourceStableID,
		})
	})
}

func (h *Handler) heartbeatLease(c *gin.Context) {
	courseBodyOK(h, c, "ok", func(courseID uint, req *leaseHeartbeatRequest) (any, error) {
		return h.svc.HeartbeatLease(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.LeaseHeartbeatInput{
			LeaseToken: req.LeaseToken,
		})
	})
}

func (h *Handler) releaseLease(c *gin.Context) {
	withCourseAndBody[leaseReleaseRequest](c, func(courseID uint, req *leaseReleaseRequest) {
		if err := h.svc.ReleaseLease(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.ReleaseLeaseInput{
			LeaseToken: req.LeaseToken,
		}); mapCourseError(c, err) {
			return
		}
		response.OK(c, "released", gin.H{"course_id": courseID})
	})
}

func toVideoContent(req *videoRequest) *domain.VideoContent {
	if req == nil {
		return nil
	}
	return &domain.VideoContent{MediaFileID: req.MediaFileID}
}

func toTextContent(req *textRequest) *domain.TextContent {
	if req == nil {
		return nil
	}
	return &domain.TextContent{ContentDelta: req.ContentDelta}
}

func toQuizContent(req *quizRequest) *domain.QuizContent {
	if req == nil {
		return nil
	}
	options := make([]domain.QuizOption, len(req.Options))
	for i, option := range req.Options {
		options[i] = domain.QuizOption{
			OptionKey: option.OptionKey,
			Body:      option.Body,
			IsCorrect: option.IsCorrect,
		}
	}
	return &domain.QuizContent{
		Prompt:        req.Prompt,
		AllowMultiple: req.AllowMultiple,
		Options:       options,
	}
}

func (h *Handler) reorderOutlineChildren(c *gin.Context, name, invalidMessage string, fn func(courseID, actorUserID, parentID uint, orderedStableIDs []string) (any, error)) {
	reorderByParent(h, c, name, invalidMessage, func(courseID, parentID uint, orderedStableIDs []string) (any, error) {
		return fn(courseID, utils.CurrentUserID(c), parentID, orderedStableIDs)
	})
}
