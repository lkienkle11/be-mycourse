package media

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/logic/utils"
)

func BindCreateFileMultipart(c *gin.Context) (dto.CreateFileRequest, error) {
	if _, err := ParseMetadataFromRaw(c.PostForm("metadata")); err != nil {
		return dto.CreateFileRequest{}, err
	}
	return dto.CreateFileRequest{
		Kind:      "",
		ObjectKey: c.PostForm("object_key"),
		Metadata:  nil,
	}, nil
}

func BindUpdateFileMultipart(c *gin.Context) (dto.UpdateFileRequest, error) {
	if _, err := ParseMetadataFromRaw(c.PostForm("metadata")); err != nil {
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
			return dto.UpdateFileRequest{}, fmt.Errorf("expected_row_version must be an integer")
		}
		req.ExpectedRowVersion = &v
	}
	return req, nil
}
