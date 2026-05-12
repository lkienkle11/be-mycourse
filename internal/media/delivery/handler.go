package delivery

import (
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/media/application"
	"mycourse-io-be/internal/media/domain" //nolint:depguard // delivery uses domain.OpenedUploadPart type; no business logic
	mediainfra "mycourse-io-be/internal/media/infra" //nolint:depguard // delivery uses infra.RequireInitialized for readiness check; TODO: move to service port
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

// Handler holds the media HTTP handlers with injected dependencies.
type Handler struct {
	svc *application.MediaService
}

// NewHandler creates a new media HTTP Handler.
func NewHandler(svc *application.MediaService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) optionsMedia(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (h *Handler) listFiles(c *gin.Context) {
	var q FileFilterRequest
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := h.svc.ListFiles(c.Request.Context(), toFilterDomain(q))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", toUploadFileResponses(rows), utils.BuildPage(q.getPage(), q.getPerPage(), total))
}

func (h *Handler) getFile(c *gin.Context) {
	objectKey := strings.TrimSpace(c.Param("id"))
	if objectKey == "" {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid object key", nil)
		return
	}
	kind := strings.TrimSpace(c.Query("kind"))
	row, err := h.svc.GetFile(c.Request.Context(), objectKey, kind)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", toUploadFileResponse(*row))
}

func (h *Handler) parseAndOpenMultipartParts(c *gin.Context) ([]domain.OpenedUploadPart, func(), error) {
	if err := c.Request.ParseMultipartForm(constants.MediaMultipartParseMemoryBytes); err != nil {
		return nil, nil, err
	}
	form := c.Request.MultipartForm
	headers := mediainfra.CollectMultipartFileHeaders(form)
	if err := mediainfra.ValidateMultipartFileHeaders(headers); err != nil {
		return nil, nil, err
	}
	parts, err := mediainfra.OpenUploadParts(headers)
	if err != nil {
		return nil, nil, err
	}
	return parts, func() { mediainfra.CloseOpenedUploadParts(parts) }, nil
}

func (h *Handler) openMultipartForMutation(c *gin.Context) ([]domain.OpenedUploadPart, func(), bool) {
	parts, closer, err := h.parseAndOpenMultipartParts(c)
	if err != nil {
		if respondMultipartValidationError(c, err) {
			return nil, nil, false
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return nil, nil, false
	}
	return parts, closer, true
}

func (h *Handler) createFile(c *gin.Context) {
	parts, closer, ok := h.openMultipartForMutation(c)
	if !ok {
		return
	}
	if closer != nil {
		defer closer()
	}
	req, err := bindCreateFileMultipart(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	rows, err := h.svc.CreateFiles(c.Request.Context(), req, parts)
	if err != nil {
		if respondMediaMutationError(c, err, true) {
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", toUploadFileResponsesFromPointers(rows))
}

func (h *Handler) updateFile(c *gin.Context) {
	objectKey := strings.TrimSpace(c.Param("id"))
	if objectKey == "" {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid object key", nil)
		return
	}
	parts, closer, ok := h.openMultipartForMutation(c)
	if !ok {
		return
	}
	if closer != nil {
		defer closer()
	}
	updateReq, createReq, err := bindUpdateAndCreateMultipart(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	rows, err := h.svc.UpdateFileBundle(c.Request.Context(), objectKey, updateReq, createReq, parts)
	if err != nil {
		if respondMediaMutationError(c, err, false) {
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", toUploadFileResponsesFromPointers(rows))
}

func (h *Handler) batchDeleteMediaFiles(c *gin.Context) {
	var req BatchDeleteMediaFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	keys := req.ObjectKeys
	if len(keys) == 0 {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, "object_keys must contain at least one object_key", nil)
		return
	}
	if len(keys) > constants.MaxMediaBatchDelete {
		response.Fail(c, http.StatusBadRequest, apperrors.MediaBatchDeleteTooManyIDs, apperrors.DefaultMessage(apperrors.MediaBatchDeleteTooManyIDs), nil)
		return
	}
	if err := h.svc.DeleteFilesByObjectKeys(c.Request.Context(), keys); err != nil {
		if respondBatchDeleteError(c, err) {
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", BatchDeleteMediaFilesResponse{DeletedCount: len(keys)})
}

func (h *Handler) deleteFile(c *gin.Context) {
	objectKey := strings.TrimSpace(c.Param("id"))
	if objectKey == "" {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid object key", nil)
		return
	}
	meta, err := mediainfra.ParseMetadataFromRaw(c.Query("metadata"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	if err := h.svc.DeleteFile(c.Request.Context(), objectKey, meta); err != nil {
		if stderrors.Is(err, apperrors.ErrDependencyNotConfigured) {
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func (h *Handler) decodeLocalURL(c *gin.Context) {
	token := c.Param("token")
	objectKey, err := mediainfra.DecodeLocalURLToken(token)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", LocalURLDecodeResponse{ObjectKey: objectKey})
}

func (h *Handler) getVideoStatus(c *gin.Context) {
	videoGUID := strings.TrimSpace(c.Param("id"))
	if videoGUID == "" {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid video guid", nil)
		return
	}
	out, err := h.svc.GetVideoStatus(c.Request.Context(), videoGUID)
	if err != nil {
		if stderrors.Is(err, apperrors.ErrDependencyNotConfigured) {
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
			return
		}
		if pe, ok := asProviderError(err); ok {
			msg := pe.Error()
			if strings.TrimSpace(msg) == "" {
				msg = apperrors.DefaultMessage(pe.Code)
			}
			response.Fail(c, httpStatusForProviderCode(pe.Code), pe.Code, msg, nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", VideoStatusResponse{Status: out.Status})
}

func (h *Handler) getMediaCleanupMetrics(c *gin.Context) {
	d, f, r := h.svc.PendingCloudCleanupCounters()
	response.OK(c, "ok", MediaCleanupMetricsResponse{
		CleanupCloudDeleted: d, CleanupCloudFailed: f, CleanupCloudRetried: r,
	})
}
