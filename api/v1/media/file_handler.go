package media

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/logic/helper"
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
	kind := constants.FileKind(strings.TrimSpace(c.Query("kind")))
	row, err := mediaservice.GetFile(objectKey, kind)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", mapping.ToUploadFileResponse(*row))
}

func createFile(c *gin.Context) {
	upload, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "file is required (multipart field: file)", nil)
		return
	}
	// Early reject when the client declared Content-Length / part size (avoids pointless buffering).
	if upload.Size >= 0 && upload.Size > constants.MaxMediaUploadFileBytes {
		response.Fail(c, http.StatusRequestEntityTooLarge, errcode.FileTooLarge, errcode.DefaultMessage(errcode.FileTooLarge), nil)
		return
	}
	file, err := upload.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, "cannot open uploaded file", nil)
		return
	}
	defer file.Close()

	meta, err := helper.ParseMetadataFromRaw(c.PostForm("metadata"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	req := dto.CreateFileRequest{
		Kind:      c.PostForm("kind"),
		ObjectKey: c.PostForm("object_key"),
		Metadata:  meta,
	}
	row, err := mediaservice.CreateFile(req, file, upload)
	if err != nil {
		if errors.Is(err, helper.ErrDependencyNotConfigured) {
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
			return
		}
		if errors.Is(err, pkgmedia.ErrFileExceedsMaxUploadSize) {
			response.Fail(c, http.StatusRequestEntityTooLarge, errcode.FileTooLarge, errcode.DefaultMessage(errcode.FileTooLarge), nil)
			return
		}
		if pe, ok := pkgmedia.AsProviderError(err); ok {
			msg := pe.Error()
			if strings.TrimSpace(msg) == "" {
				msg = errcode.DefaultMessage(pe.Code)
			}
			response.Fail(c, pkgmedia.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
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
	defer file.Close()

	meta, err := helper.ParseMetadataFromRaw(c.PostForm("metadata"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	req := dto.UpdateFileRequest{
		Kind:     c.PostForm("kind"),
		Metadata: meta,
	}
	row, err := mediaservice.UpdateFile(objectKey, req, file, upload)
	if err != nil {
		if errors.Is(err, helper.ErrDependencyNotConfigured) {
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
			return
		}
		if errors.Is(err, pkgmedia.ErrFileExceedsMaxUploadSize) {
			response.Fail(c, http.StatusRequestEntityTooLarge, errcode.FileTooLarge, errcode.DefaultMessage(errcode.FileTooLarge), nil)
			return
		}
		if pe, ok := pkgmedia.AsProviderError(err); ok {
			msg := pe.Error()
			if strings.TrimSpace(msg) == "" {
				msg = errcode.DefaultMessage(pe.Code)
			}
			response.Fail(c, pkgmedia.HTTPStatusForProviderCode(pe.Code), pe.Code, msg, nil)
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
	meta, err := helper.ParseMetadataFromRaw(c.Query("metadata"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	if err := mediaservice.DeleteFile(objectKey, meta); err != nil {
		if errors.Is(err, helper.ErrDependencyNotConfigured) {
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
	objectKey, err := helper.DecodeLocalURLToken(token)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.BadRequest, err.Error(), nil)
		return
	}
	response.OK(c, "ok", gin.H{"object_key": objectKey})
}
