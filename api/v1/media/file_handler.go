package media

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
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

// openMultipartFileField binds one multipart file and enforces maxBytes when Size is known.
func openMultipartFileField(c *gin.Context, field string, maxBytes int64) (multipart.File, *multipart.FileHeader, bool) {
	upload, err := c.FormFile(field)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "file is required (multipart field: "+field+")", nil)
		return nil, nil, false
	}
	if upload.Size >= 0 && upload.Size > maxBytes {
		response.Fail(c, http.StatusRequestEntityTooLarge, errcode.FileTooLarge, errcode.DefaultMessage(errcode.FileTooLarge), nil)
		return nil, nil, false
	}
	file, err := upload.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "cannot open uploaded file", nil)
		return nil, nil, false
	}
	return file, upload, true
}

func createFile(c *gin.Context) {
	upload, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "file is required (multipart field: file)", nil)
		return
	}
	if upload.Size >= 0 && upload.Size > constants.MaxMediaUploadFileBytes {
		response.Fail(c, http.StatusRequestEntityTooLarge, errcode.FileTooLarge, errcode.DefaultMessage(errcode.FileTooLarge), nil)
		return
	}
	file, err := upload.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "cannot open uploaded file", nil)
		return
	}
	defer func() { _ = file.Close() }()

	req, err := pkgmedia.BindCreateFileMultipart(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	row, err := mediaservice.CreateFile(req, file, upload)
	if err != nil {
		if respondMediaMutationError(c, err, true) {
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.Created(c, "created", mapping.ToUploadFileResponse(*row))
}

func updateFile(c *gin.Context) {
	objectKey := strings.TrimSpace(c.Param("id"))
	if objectKey == "" {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "invalid object key", nil)
		return
	}

	file, upload, ok := openMultipartFileField(c, "file", constants.MaxMediaUploadFileBytes)
	if !ok {
		return
	}
	defer func() { _ = file.Close() }()

	req, err := pkgmedia.BindUpdateFileMultipart(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	row, err := mediaservice.UpdateFile(objectKey, req, file, upload)
	if err != nil {
		if respondMediaMutationError(c, err, false) {
			return
		}
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "updated", mapping.ToUploadFileResponse(*row))
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
	response.OK(c, "ok", gin.H{"object_key": objectKey})
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
	response.OK(c, "ok", out)
}

func getMediaCleanupMetrics(c *gin.Context) {
	response.OK(c, "ok", gin.H{
		"cleanup_cloud_deleted": mediaservice.CleanupCloudDeleted.Load(),
		"cleanup_cloud_failed":  mediaservice.CleanupCloudFailed.Load(),
		"cleanup_cloud_retried": mediaservice.CleanupCloudRetried.Load(),
	})
}
