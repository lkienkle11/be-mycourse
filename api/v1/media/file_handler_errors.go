package media

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/response"
)

// respondMediaMutationError maps create/update file service errors to HTTP responses.
// When includeExecutableReject is true, ErrExecutableUploadRejected maps to 400 + ExecutableUploadRejected.
// Returns true if err was handled (caller should return).
func respondMediaMutationError(c *gin.Context, err error, includeExecutableReject bool) bool {
	if errors.Is(err, pkgerrors.ErrDependencyNotConfigured) {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrFileExceedsMaxUploadSize) {
		response.Fail(c, http.StatusRequestEntityTooLarge, errcode.FileTooLarge, errcode.DefaultMessage(errcode.FileTooLarge), nil)
		return true
	}
	if includeExecutableReject && errors.Is(err, pkgerrors.ErrExecutableUploadRejected) {
		response.Fail(c, http.StatusBadRequest, errcode.ExecutableUploadRejected, errcode.DefaultMessage(errcode.ExecutableUploadRejected), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrMediaOptimisticLock) || errors.Is(err, pkgerrors.ErrMediaReuseMismatch) {
		response.Fail(c, http.StatusConflict, errcode.Conflict, err.Error(), nil)
		return true
	}
	if pe, ok := pkgerrors.AsProviderError(err); ok {
		msg := pe.Error()
		if strings.TrimSpace(msg) == "" {
			msg = errcode.DefaultMessage(pe.Code)
		}
		response.Fail(c, pkgerrors.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
		return true
	}
	return false
}
