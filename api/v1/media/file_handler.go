package media

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	jobmedia "mycourse-io-be/internal/jobs/media"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/mapping"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/response"
	mediaservice "mycourse-io-be/services/media"
)

func optionsMedia(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func listFiles(c *gin.Context) {
	var q dto.FileFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}
	rows, total, err := mediaservice.ListFiles(q)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		return
	}
	response.OKPaginated(c, "ok", mapping.ToUploadFileResponses(rows), utils.BuildPage(q.GetPage(), q.GetPerPage(), total))
}

func getFile(c *gin.Context) {
	objectKey := strings.TrimSpace(c.Param("id"))
	if objectKey == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid object key", nil)
		return
	}
	kind := string(strings.TrimSpace(c.Query("kind")))
	row, err := mediaservice.GetFile(objectKey, kind)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", mapping.ToUploadFileResponse(*row))
}

func parseAndOpenMultipartParts(c *gin.Context) ([]entities.OpenedUploadPart, func(), error) {
	if err := c.Request.ParseMultipartForm(constants.MediaMultipartParseMemoryBytes); err != nil {
		return nil, nil, err
	}
	form := c.Request.MultipartForm
	headers := pkgmedia.CollectMultipartFileHeaders(form)
	if err := pkgmedia.ValidateMultipartFileHeaders(headers); err != nil {
		return nil, nil, err
	}
	parts, err := pkgmedia.OpenUploadParts(headers)
	if err != nil {
		return nil, nil, err
	}
	closer := func() { pkgmedia.CloseOpenedUploadParts(parts) }
	return parts, closer, nil
}

func bindUpdateAndCreateMultipart(c *gin.Context) (dto.UpdateFileRequest, dto.CreateFileRequest, error) {
	updateReq, err := mapping.BindUpdateFileMultipart(c)
	if err != nil {
		return dto.UpdateFileRequest{}, dto.CreateFileRequest{}, err
	}
	createReq, err := mapping.BindCreateFileMultipart(c)
	return updateReq, createReq, err
}

func openMultipartForMutation(c *gin.Context) ([]entities.OpenedUploadPart, func(), bool) {
	parts, closer, err := parseAndOpenMultipartParts(c)
	if err != nil {
		if respondMultipartValidationError(c, err) {
			return nil, nil, false
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return nil, nil, false
	}
	return parts, closer, true
}

func createFile(c *gin.Context) {
	parts, closer, ok := openMultipartForMutation(c)
	if !ok {
		return
	}
	if closer != nil {
		defer closer()
	}

	req, err := mapping.BindCreateFileMultipart(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	rows, err := mediaservice.CreateFiles(req, parts)
	if err != nil {
		if respondMediaMutationError(c, err, true) {
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", mapping.ToUploadFileResponsesFromPointers(rows))
}

func updateFile(c *gin.Context) {
	objectKey := strings.TrimSpace(c.Param("id"))
	if objectKey == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid object key", nil)
		return
	}

	parts, closer, ok := openMultipartForMutation(c)
	if !ok {
		return
	}
	if closer != nil {
		defer closer()
	}

	updateReq, createReq, err := bindUpdateAndCreateMultipart(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	rows, err := mediaservice.UpdateFileBundle(objectKey, updateReq, createReq, parts)
	if err != nil {
		if respondMediaMutationError(c, err, false) {
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", mapping.ToUploadFileResponsesFromPointers(rows))
}

func batchDeleteMediaFiles(c *gin.Context) {
	var req dto.BatchDeleteMediaFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}
	keys := req.ObjectKeys
	if len(keys) == 0 {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, "object_keys must contain at least one object_key", nil)
		return
	}
	if len(keys) > constants.MaxMediaBatchDelete {
		response.Fail(c, http.StatusBadRequest, errcode.MediaBatchDeleteTooManyIDs, errcode.DefaultMessage(errcode.MediaBatchDeleteTooManyIDs), nil)
		return
	}
	if err := mediaservice.DeleteFilesByObjectKeys(keys); err != nil {
		if respondBatchDeleteError(c, err) {
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", dto.BatchDeleteMediaFilesResponse{DeletedCount: len(keys)})
}

func deleteFile(c *gin.Context) {
	objectKey := strings.TrimSpace(c.Param("id"))
	if objectKey == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid object key", nil)
		return
	}
	meta, err := pkgmedia.ParseMetadataFromRaw(c.Query("metadata"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	if err := mediaservice.DeleteFile(objectKey, meta); err != nil {
		if errors.Is(err, pkgerrors.ErrDependencyNotConfigured) {
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "deleted", nil)
}

func decodeLocalURL(c *gin.Context) {
	token := c.Param("token")
	objectKey, err := pkgmedia.DecodeLocalURLToken(token)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", dto.LocalURLDecodeResponse{ObjectKey: objectKey})
}

func getVideoStatus(c *gin.Context) {
	videoGUID := strings.TrimSpace(c.Param("id"))
	if videoGUID == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid video guid", nil)
		return
	}
	out, err := mediaservice.GetVideoStatus(c.Request.Context(), videoGUID)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrDependencyNotConfigured) {
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
			return
		}
		if pe, ok := pkgerrors.AsProviderError(err); ok {
			msg := pe.Error()
			if strings.TrimSpace(msg) == "" {
				msg = errcode.DefaultMessage(pe.Code)
			}
			response.Fail(c, pkgerrors.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", dto.VideoStatusResponse{Status: out.Status})
}

func getMediaCleanupMetrics(c *gin.Context) {
	response.OK(c, "ok", dto.MediaCleanupMetricsResponse{
		CleanupCloudDeleted: jobmedia.CleanupCloudDeleted.Load(),
		CleanupCloudFailed:  jobmedia.CleanupCloudFailed.Load(),
		CleanupCloudRetried: jobmedia.CleanupCloudRetried.Load(),
	})
}
