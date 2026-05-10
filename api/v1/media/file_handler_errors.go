package media

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	errfuncmedia "mycourse-io-be/pkg/errors_func/media"
	"mycourse-io-be/pkg/response"
)

// respondMultipartValidationError maps declarative multipart validation errors (count/total size).
func respondMultipartValidationError(c *gin.Context, err error) bool {
	if errors.Is(err, pkgerrors.ErrMediaFilesRequired) {
		response.Fail(c, http.StatusBadRequest, errcode.MediaFilesRequired, errcode.DefaultMessage(errcode.MediaFilesRequired), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrMediaTooManyFilesInRequest) {
		response.Fail(c, http.StatusBadRequest, errcode.MediaTooManyFilesInRequest, errcode.DefaultMessage(errcode.MediaTooManyFilesInRequest), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrMediaMultipartTotalTooLarge) {
		response.Fail(c, http.StatusRequestEntityTooLarge, errcode.MediaMultipartTotalTooLarge, errcode.DefaultMessage(errcode.MediaMultipartTotalTooLarge), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrFileExceedsMaxUploadSize) {
		response.Fail(c, http.StatusRequestEntityTooLarge, errcode.FileTooLarge, errcode.DefaultMessage(errcode.FileTooLarge), nil)
		return true
	}
	return false
}

// respondMediaMutationError maps create/update file service errors to HTTP responses.
// When includeExecutableReject is true, ErrExecutableUploadRejected maps to 400 + ExecutableUploadRejected.
// Returns true if err was handled (caller should return).
func respondMediaMutationError(c *gin.Context, err error, includeExecutableReject bool) bool {
	if respondMultipartValidationError(c, err) {
		return true
	}
	if errors.Is(err, pkgerrors.ErrDependencyNotConfigured) {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
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
	if pe, ok := errfuncmedia.AsProviderError(err); ok {
		msg := pe.Error()
		if strings.TrimSpace(msg) == "" {
			msg = errcode.DefaultMessage(pe.Code)
		}
		response.Fail(c, errfuncmedia.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
		return true
	}
	return false
}

// respondBatchDeleteError maps batch delete service errors to HTTP responses.
func respondBatchDeleteError(c *gin.Context, err error) bool {
	if errors.Is(err, pkgerrors.ErrBatchDeleteEmptyKeys) {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrMediaBatchDeleteTooManyIDs) {
		response.Fail(c, http.StatusBadRequest, errcode.MediaBatchDeleteTooManyIDs, errcode.DefaultMessage(errcode.MediaBatchDeleteTooManyIDs), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrMediaDuplicateObjectKeysInBatchDelete) {
		response.Fail(c, http.StatusBadRequest, errcode.MediaDuplicateKeysInBatchDelete, errcode.DefaultMessage(errcode.MediaDuplicateKeysInBatchDelete), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrMediaObjectKeyRequired) {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrMediaFileNotFoundForObjectKey) {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return true
	}
	if errors.Is(err, pkgerrors.ErrDependencyNotConfigured) {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return true
	}
	return false
}
