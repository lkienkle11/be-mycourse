package mapping

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
)

// BindCreateFileMultipart reads optional multipart text fields into dto.CreateFileRequest (legacy multipart bind path).
func BindCreateFileMultipart(c *gin.Context) (dto.CreateFileRequest, error) {
	if _, err := pkgmedia.ParseMetadataFromRaw(c.PostForm("metadata")); err != nil {
		return dto.CreateFileRequest{}, err
	}
	return dto.CreateFileRequest{
		Kind:      "",
		ObjectKey: c.PostForm("object_key"),
		Metadata:  nil,
	}, nil
}

// BindUpdateFileMultipart reads optional multipart text fields into dto.UpdateFileRequest (legacy multipart bind path).
func BindUpdateFileMultipart(c *gin.Context) (dto.UpdateFileRequest, error) {
	if _, err := pkgmedia.ParseMetadataFromRaw(c.PostForm("metadata")); err != nil {
		return dto.UpdateFileRequest{}, err
	}
	req := dto.UpdateFileRequest{
		Kind:                  "",
		Metadata:              nil,
		ReuseMediaID:          strings.TrimSpace(c.PostForm("reuse_media_id")),
		SkipUploadIfUnchanged: utils.ParseBoolLoose(c.PostForm("skip_upload_if_unchanged")),
	}
	if ev := strings.TrimSpace(c.PostForm("expected_row_version")); ev != "" {
		v, perr := strconv.ParseInt(ev, 10, 64)
		if perr != nil {
			return dto.UpdateFileRequest{}, fmt.Errorf("%s", constants.MsgExpectedRowVersionInteger)
		}
		req.ExpectedRowVersion = &v
	}
	return req, nil
}
