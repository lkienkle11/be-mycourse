package delivery

import (
	"context"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/application"
	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

func (h *Handler) createSection(c *gin.Context) {
	courseBodyCreated(h, c, "created", func(courseID string, req *sectionRequest) (any, error) {
		return h.svc.CreateSection(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.UpsertSectionInput{
			Title:       req.Title,
			Description: req.Description,
		})
	})
}

func (h *Handler) updateSection(c *gin.Context) {
	courseParamBodyOK(h, c, "sectionId", "invalid section id", "updated", func(courseID, sectionID string, req *sectionRequest) (any, error) {
		return h.svc.UpdateSection(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.UpsertSectionInput{
			SectionID:          &sectionID,
			ExpectedRowVersion: req.ExpectedRowVersion,
			Title:              req.Title,
			Description:        req.Description,
		})
	})
}

func (h *Handler) deleteSection(c *gin.Context) {
	h.courseParamOK(c, "sectionId", "invalid section id", "deleted", func(courseID, sectionID string) (any, error) {
		return h.svc.DeleteSection(c.Request.Context(), courseID, utils.CurrentUserID(c), sectionID)
	})
}

func (h *Handler) reorderSections(c *gin.Context) {
	courseBodyUpdated(h, c, func(courseID string, req *reorderRequest) (any, error) {
		return h.svc.ReorderSections(c.Request.Context(), courseID, utils.CurrentUserID(c), req.OrderedStableIDs)
	})
}

func (h *Handler) createLesson(c *gin.Context) { h.upsertLesson(c, true) }
func (h *Handler) updateLesson(c *gin.Context) { h.upsertLesson(c, false) }

func (h *Handler) upsertLesson(c *gin.Context, create bool) {
	handler := func(courseID string, lessonID *string, req *lessonRequest) {
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
		withCourseAndBody[lessonRequest](c, func(courseID string, req *lessonRequest) {
			handler(courseID, nil, req)
		})
		return
	}

	withCourseParamAndBody[lessonRequest](c, "lessonId", "invalid lesson id", func(courseID, lessonID string, req *lessonRequest) {
		handler(courseID, &lessonID, req)
	})
}

func (h *Handler) deleteLesson(c *gin.Context) {
	h.courseParamOK(c, "lessonId", "invalid lesson id", "deleted", func(courseID, lessonID string) (any, error) {
		return h.svc.DeleteLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), lessonID)
	})
}

func (h *Handler) reorderLessons(c *gin.Context) {
	h.reorderOutline(c, reorderSpec{parentParam: "sectionId", invalidMessage: "invalid section id", call: reorderLessonsCall})
}

func (h *Handler) createSubLesson(c *gin.Context) { h.upsertSubLesson(c, true) }
func (h *Handler) updateSubLesson(c *gin.Context) { h.upsertSubLesson(c, false) }

func (h *Handler) upsertSubLesson(c *gin.Context, create bool) {
	handler := func(courseID string, subLessonID *string, req *subLessonRequest) {
		estimatedMs := int64(0)
		if req.EstimatedDurationMs != nil {
			estimatedMs = *req.EstimatedDurationMs
		}
		input := domain.UpsertSubLessonInput{
			SubLessonID:         subLessonID,
			LessonID:            req.LessonID,
			ExpectedRowVersion:  req.ExpectedRowVersion,
			Title:               req.Title,
			Kind:                req.Kind,
			IsPreview:           req.IsPreview,
			EstimatedDurationMs: estimatedMs,
			Video:               toVideoContent(req.Video),
			Text:                toTextContent(req.Text),
			Quiz:                toQuizContent(req.Quiz),
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
		withCourseAndBody[subLessonRequest](c, func(courseID string, req *subLessonRequest) {
			handler(courseID, nil, req)
		})
		return
	}

	withCourseParamAndBody[subLessonRequest](c, "subLessonId", "invalid sub-lesson id", func(courseID, subLessonID string, req *subLessonRequest) {
		handler(courseID, &subLessonID, req)
	})
}

func (h *Handler) deleteSubLesson(c *gin.Context) {
	h.courseParamOK(c, "subLessonId", "invalid sub-lesson id", "deleted", func(courseID, subLessonID string) (any, error) {
		return h.svc.DeleteSubLesson(c.Request.Context(), courseID, utils.CurrentUserID(c), subLessonID)
	})
}

func (h *Handler) reorderSubLessons(c *gin.Context) {
	h.reorderOutline(c, reorderSpec{parentParam: "lessonId", invalidMessage: "invalid lesson id", call: reorderSubLessonsCall})
}

func (h *Handler) acquireLease(c *gin.Context) {
	courseBodyOK(h, c, "ok", func(courseID string, req *leaseAcquireRequest) (any, error) {
		return h.svc.AcquireLease(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.AcquireLeaseInput{
			CourseVersionID:  req.CourseVersionID,
			ResourceType:     req.ResourceType,
			ResourceStableID: req.ResourceStableID,
		})
	})
}

func (h *Handler) heartbeatLease(c *gin.Context) {
	courseBodyOK(h, c, "ok", func(courseID string, req *leaseHeartbeatRequest) (any, error) {
		return h.svc.HeartbeatLease(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.LeaseHeartbeatInput{
			LeaseToken: req.LeaseToken,
		})
	})
}

func (h *Handler) releaseLease(c *gin.Context) {
	withCourseAndBody[leaseReleaseRequest](c, func(courseID string, req *leaseReleaseRequest) {
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

func (h *Handler) reorderOutlineChildren(c *gin.Context, name, invalidMessage string, fn func(courseID string, actorUserID string, parentID string, orderedStableIDs []string) (any, error)) {
	reorderByParent(h, c, name, invalidMessage, func(courseID, parentID string, orderedStableIDs []string) (any, error) {
		return fn(courseID, utils.CurrentUserID(c), parentID, orderedStableIDs)
	})
}

type reorderSpec struct {
	parentParam    string
	invalidMessage string
	call           func(svc *application.CourseService, ctx context.Context, courseID, actorUserID, parentID string, orderedStableIDs []string) (any, error)
}

func (h *Handler) reorderOutline(c *gin.Context, spec reorderSpec) {
	h.reorderOutlineChildren(c, spec.parentParam, spec.invalidMessage, func(courseID string, actorUserID string, parentID string, orderedStableIDs []string) (any, error) {
		return spec.call(h.svc, c.Request.Context(), courseID, actorUserID, parentID, orderedStableIDs)
	})
}

func reorderLessonsCall(svc *application.CourseService, ctx context.Context, courseID, actorUserID, parentID string, orderedStableIDs []string) (any, error) {
	return svc.ReorderLessons(ctx, courseID, actorUserID, parentID, orderedStableIDs)
}

func reorderSubLessonsCall(svc *application.CourseService, ctx context.Context, courseID, actorUserID, parentID string, orderedStableIDs []string) (any, error) {
	return svc.ReorderSubLessons(ctx, courseID, actorUserID, parentID, orderedStableIDs)
}
