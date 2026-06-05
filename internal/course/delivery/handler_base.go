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
		domain.ErrCourseLeaseTokenInvalid:
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
	case apperrors.ErrNotFound, apperrors.ErrInvalidProfileMediaFile:
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
	default:
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
	}
	return true
}

func currentUserID(c *gin.Context) uint { return utils.CurrentUserID(c) }

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

func withCourseID(c *gin.Context, fn func(courseID uint)) {
	courseID, ok := utils.ParseUintParam(c, "courseId")
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

func withCourseAndBody[T any](c *gin.Context, fn func(courseID uint, req *T)) {
	withCourseID(c, func(courseID uint) {
		withBody(c, func(req *T) {
			fn(courseID, req)
		})
	})
}

func withCourseAndParam(c *gin.Context, name, message string, fn func(courseID, id uint)) {
	withCourseID(c, func(courseID uint) {
		id, ok := utils.ParseUintParam(c, name)
		if !ok {
			badParam(c, message)
			return
		}
		fn(courseID, id)
	})
}

func withCourseParamAndBody[T any](c *gin.Context, name, message string, fn func(courseID, id uint, req *T)) {
	withCourseAndParam(c, name, message, func(courseID, id uint) {
		withBody(c, func(req *T) {
			fn(courseID, id, req)
		})
	})
}

func writeResponse(c *gin.Context, statusCode int, message string, payload any) {
	response.WriteByStatus(c, statusCode, message, payload)
}

func courseResult(h *Handler, c *gin.Context, statusCode int, message string, fn func(courseID uint) (any, error)) {
	withCourseID(c, func(courseID uint) {
		row, err := fn(courseID)
		if mapCourseError(c, err) {
			return
		}
		writeResponse(c, statusCode, message, row)
	})
}

func (h *Handler) courseOK(c *gin.Context, message string, fn func(courseID uint) (any, error)) {
	courseResult(h, c, http.StatusOK, message, fn)
}

func (h *Handler) courseCreated(c *gin.Context, message string, fn func(courseID uint) (any, error)) {
	courseResult(h, c, http.StatusCreated, message, fn)
}

func (h *Handler) courseParamOK(c *gin.Context, name, invalidMessage, message string, fn func(courseID, id uint) (any, error)) {
	withCourseAndParam(c, name, invalidMessage, func(courseID, id uint) {
		row, err := fn(courseID, id)
		if mapCourseError(c, err) {
			return
		}
		response.OK(c, message, row)
	})
}

func courseBodyOK[T any](h *Handler, c *gin.Context, message string, fn func(courseID uint, req *T) (any, error)) {
	courseBodyResult(h, c, http.StatusOK, message, fn)
}

func courseBodyCreated[T any](h *Handler, c *gin.Context, message string, fn func(courseID uint, req *T) (any, error)) {
	courseBodyResult(h, c, http.StatusCreated, message, fn)
}

func courseBodyResult[T any](h *Handler, c *gin.Context, statusCode int, message string, fn func(courseID uint, req *T) (any, error)) {
	withCourseAndBody[T](c, func(courseID uint, req *T) {
		row, err := fn(courseID, req)
		if mapCourseError(c, err) {
			return
		}
		writeResponse(c, statusCode, message, row)
	})
}

func courseParamBodyOK[T any](h *Handler, c *gin.Context, name, invalidMessage, message string, fn func(courseID, id uint, req *T) (any, error)) {
	withCourseParamAndBody[T](c, name, invalidMessage, func(courseID, id uint, req *T) {
		row, err := fn(courseID, id, req)
		if mapCourseError(c, err) {
			return
		}
		response.OK(c, message, row)
	})
}

func reorderByParent(h *Handler, c *gin.Context, name, invalidMessage string, fn func(courseID, parentID uint, orderedStableIDs []string) (any, error)) {
	courseParamBodyOK(h, c, name, invalidMessage, "updated", func(courseID, parentID uint, req *reorderRequest) (any, error) {
		return fn(courseID, parentID, req.OrderedStableIDs)
	})
}
