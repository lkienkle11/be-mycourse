package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
)

func parseBoolLoose(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "1" || v == "true" || v == "yes"
}

func BindCreateFileMultipart(c *gin.Context) (dto.CreateFileRequest, error) {
	meta, err := ParseMetadataFromRaw(c.PostForm("metadata"))
	if err != nil {
		return dto.CreateFileRequest{}, err
	}
	return dto.CreateFileRequest{
		Kind:      c.PostForm("kind"),
		ObjectKey: c.PostForm("object_key"),
		Metadata:  meta,
	}, nil
}

func BindUpdateFileMultipart(c *gin.Context) (dto.UpdateFileRequest, error) {
	meta, err := ParseMetadataFromRaw(c.PostForm("metadata"))
	if err != nil {
		return dto.UpdateFileRequest{}, err
	}
	req := dto.UpdateFileRequest{
		Kind:                  c.PostForm("kind"),
		Metadata:              meta,
		ReuseMediaID:          strings.TrimSpace(c.PostForm("reuse_media_id")),
		SkipUploadIfUnchanged: parseBoolLoose(c.PostForm("skip_upload_if_unchanged")),
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
