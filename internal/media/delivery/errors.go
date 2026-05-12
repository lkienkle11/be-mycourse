package delivery

import (
	stderrors "errors"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/media/domain" //nolint:depguard // delivery unwraps domain.ProviderError for response mapping; no business logic
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
)

var (
	errBunnyWebhookJSONInvalid    = stderrors.New("bunny webhook JSON invalid")
	errBunnyWebhookPayloadInvalid = stderrors.New("bunny webhook payload invalid")
)

var jsonUnmarshal = json.Unmarshal

// asProviderError unwraps *domain.ProviderError from err.
func asProviderError(err error) (*domain.ProviderError, bool) {
	var pe *domain.ProviderError
	if stderrors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

// httpStatusForProviderCode maps provider error codes to HTTP status codes.
func httpStatusForProviderCode(code int) int {
	switch code {
	case apperrors.BunnyCreateFailed, apperrors.BunnyUploadFailed, apperrors.BunnyInvalidResponse:
		return http.StatusBadGateway
	case apperrors.BunnyVideoNotFound:
		return http.StatusNotFound
	case apperrors.BunnyGetVideoFailed:
		return http.StatusBadGateway
	case apperrors.ImageEncodeBusy:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// respondMultipartValidationError maps multipart size/count errors to HTTP responses.
func respondMultipartValidationError(c *gin.Context, err error) bool {
	if stderrors.Is(err, apperrors.ErrMediaFilesRequired) {
		response.Fail(c, http.StatusBadRequest, apperrors.MediaFilesRequired, apperrors.DefaultMessage(apperrors.MediaFilesRequired), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrMediaTooManyFilesInRequest) {
		response.Fail(c, http.StatusBadRequest, apperrors.MediaTooManyFilesInRequest, apperrors.DefaultMessage(apperrors.MediaTooManyFilesInRequest), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrMediaMultipartTotalTooLarge) {
		response.Fail(c, http.StatusRequestEntityTooLarge, apperrors.MediaMultipartTotalTooLarge, apperrors.DefaultMessage(apperrors.MediaMultipartTotalTooLarge), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrFileExceedsMaxUploadSize) {
		response.Fail(c, http.StatusRequestEntityTooLarge, apperrors.FileTooLarge, apperrors.DefaultMessage(apperrors.FileTooLarge), nil)
		return true
	}
	return false
}

// respondMediaMutationError maps create/update errors to HTTP.
func respondMediaMutationError(c *gin.Context, err error, includeExecutableReject bool) bool {
	if respondMultipartValidationError(c, err) {
		return true
	}
	if stderrors.Is(err, apperrors.ErrDependencyNotConfigured) {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return true
	}
	if includeExecutableReject && stderrors.Is(err, apperrors.ErrExecutableUploadRejected) {
		response.Fail(c, http.StatusBadRequest, apperrors.ExecutableUploadRejected, apperrors.DefaultMessage(apperrors.ExecutableUploadRejected), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrMediaOptimisticLock) || stderrors.Is(err, apperrors.ErrMediaReuseMismatch) {
		response.Fail(c, http.StatusConflict, apperrors.Conflict, err.Error(), nil)
		return true
	}
	if pe, ok := asProviderError(err); ok {
		msg := pe.Error()
		if strings.TrimSpace(msg) == "" {
			msg = apperrors.DefaultMessage(pe.Code)
		}
		response.Fail(c, httpStatusForProviderCode(pe.Code), pe.Code, msg, nil)
		return true
	}
	return false
}

// respondBatchDeleteError maps batch delete errors to HTTP.
func respondBatchDeleteError(c *gin.Context, err error) bool {
	if stderrors.Is(err, apperrors.ErrBatchDeleteEmptyKeys) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrMediaBatchDeleteTooManyIDs) {
		response.Fail(c, http.StatusBadRequest, apperrors.MediaBatchDeleteTooManyIDs, apperrors.DefaultMessage(apperrors.MediaBatchDeleteTooManyIDs), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrMediaDuplicateObjectKeysInBatchDelete) {
		response.Fail(c, http.StatusBadRequest, apperrors.MediaDuplicateKeysInBatchDelete, apperrors.DefaultMessage(apperrors.MediaDuplicateKeysInBatchDelete), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrMediaObjectKeyRequired) || stderrors.Is(err, apperrors.ErrMediaFileNotFoundForObjectKey) {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return true
	}
	if stderrors.Is(err, apperrors.ErrDependencyNotConfigured) {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return true
	}
	return false
}
