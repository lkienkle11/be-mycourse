package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/application"
	"mycourse-io-be/internal/course/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
)

type Handler struct {
	svc *application.CourseService
}

func NewHandler(svc *application.CourseService) *Handler { return &Handler{svc: svc} }

func mapCourseError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch err {
	case domain.ErrCourseNotFound, domain.ErrCourseVersionNotFound, domain.ErrCourseEnrollmentNotFound:
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, err.Error(), nil)
	case domain.ErrCourseOwnerOnly, domain.ErrCourseCollaboratorAccess:
		response.Fail(c, http.StatusForbidden, apperrors.Forbidden, err.Error(), nil)
	case domain.ErrCourseOptimisticLock, domain.ErrCourseLeaseHeldByOtherUser:
		response.Fail(c, http.StatusConflict, apperrors.Conflict, err.Error(), nil)
	case domain.ErrCourseDraftRequired, domain.ErrCourseDraftInReview, domain.ErrCourseDraftRejectedOnly,
		domain.ErrCourseInvalidSubLessonKind, domain.ErrCourseInvalidReviewState, domain.ErrCourseInvalidOrdering,
		domain.ErrCourseInstructorRequired, domain.ErrCourseOwnerCannotBeRemoved, domain.ErrCoursePublishedRequired,
		domain.ErrCourseLeaseTokenInvalid, domain.ErrCourseInvalidSlug, domain.ErrCourseTitleTooShort,
		domain.ErrCoursePreviewNotAllowedForQuiz, domain.ErrCourseSubmitBasicInfoIncomplete,
		domain.ErrCourseSubmitOutlineIncomplete, domain.ErrCourseSubmitInvalidSubLesson,
		domain.ErrCourseSubmitCollaboratorRequired, domain.ErrCourseCollaboratorInactive:
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
	case apperrors.ErrNotFound, apperrors.ErrInvalidProfileMediaFile:
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
	default:
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
	}
	return true
}

func badParam(c *gin.Context, msg string) {
	response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, msg, nil)
}

func bindJSON[T any](c *gin.Context) (*T, bool) {
	var req T
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return nil, false
	}
	return &req, true
}

func withCourseID(c *gin.Context, fn func(courseID string)) {
	courseID, ok := utils.ParseUUIDParam(c, "courseId")
	if !ok {
		badParam(c, "invalid course id")
		return
	}
	fn(courseID)
}

func withBody[T any](c *gin.Context, fn func(req *T)) {
	req, ok := bindJSON[T](c)
	if !ok {
		return
	}
	fn(req)
}

func withCourseAndBody[T any](c *gin.Context, fn func(courseID string, req *T)) {
	withCourseID(c, func(courseID string) {
		withBody(c, func(req *T) {
			fn(courseID, req)
		})
	})
}

func withCourseAndParam(c *gin.Context, name, message string, fn func(courseID, id string)) {
	withCourseID(c, func(courseID string) {
		id, ok := utils.ParseUUIDParam(c, name)
		if !ok {
			badParam(c, message)
			return
		}
		fn(courseID, id)
	})
}

func withCourseParamAndBody[T any](c *gin.Context, name, message string, fn func(courseID, id string, req *T)) {
	withCourseAndParam(c, name, message, func(courseID, id string) {
		withBody(c, func(req *T) {
			fn(courseID, id, req)
		})
	})
}

func writeResponse(c *gin.Context, statusCode int, message string, payload any) {
	response.WriteByStatus(c, statusCode, message, payload)
}

func courseResult(h *Handler, c *gin.Context, statusCode int, message string, fn func(courseID string) (any, error)) {
	withCourseID(c, func(courseID string) {
		row, err := fn(courseID)
		if mapCourseError(c, err) {
			return
		}
		writeResponse(c, statusCode, message, row)
	})
}

func (h *Handler) courseOK(c *gin.Context, message string, fn func(courseID string) (any, error)) {
	courseResult(h, c, http.StatusOK, message, fn)
}

func (h *Handler) courseCreated(c *gin.Context, message string, fn func(courseID string) (any, error)) {
	courseResult(h, c, http.StatusCreated, message, fn)
}

func (h *Handler) courseParamOK(c *gin.Context, name, invalidMessage, message string, fn func(courseID, id string) (any, error)) {
	withCourseAndParam(c, name, invalidMessage, func(courseID, id string) {
		row, err := fn(courseID, id)
		if mapCourseError(c, err) {
			return
		}
		response.OK(c, message, row)
	})
}

func (h *Handler) courseUUIDParamOK(c *gin.Context, name, invalidMessage, message string, fn func(courseID string, id string) (any, error)) {
	withCourseID(c, func(courseID string) {
		id, ok := utils.ParseUUIDParam(c, name)
		if !ok {
			badParam(c, invalidMessage)
			return
		}
		row, err := fn(courseID, id)
		if mapCourseError(c, err) {
			return
		}
		response.OK(c, message, row)
	})
}

func courseBodyOK[T any](h *Handler, c *gin.Context, message string, fn func(courseID string, req *T) (any, error)) {
	courseBodyResult(h, c, http.StatusOK, message, fn)
}

func courseBodyCreated[T any](h *Handler, c *gin.Context, message string, fn func(courseID string, req *T) (any, error)) {
	courseBodyResult(h, c, http.StatusCreated, message, fn)
}

func courseBodyResult[T any](h *Handler, c *gin.Context, statusCode int, message string, fn func(courseID string, req *T) (any, error)) {
	withCourseAndBody[T](c, func(courseID string, req *T) {
		row, err := fn(courseID, req)
		if mapCourseError(c, err) {
			return
		}
		writeResponse(c, statusCode, message, row)
	})
}

func courseParamBodyOK[T any](h *Handler, c *gin.Context, name, invalidMessage, message string, fn func(courseID, id string, req *T) (any, error)) {
	withCourseParamAndBody[T](c, name, invalidMessage, func(courseID, id string, req *T) {
		row, err := fn(courseID, id, req)
		if mapCourseError(c, err) {
			return
		}
		response.OK(c, message, row)
	})
}

func reorderByParent(h *Handler, c *gin.Context, name, invalidMessage string, fn func(courseID, parentID string, orderedStableIDs []string) (any, error)) {
	courseParamBodyOK(h, c, name, invalidMessage, "updated", func(courseID, parentID string, req *reorderRequest) (any, error) {
		return fn(courseID, parentID, req.OrderedStableIDs)
	})
}
