package delivery

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/media/application"
	"mycourse-io-be/internal/media/domain"           //nolint:depguard // delivery maps domain.File → DTO; pure data transformation, no business logic
	mediainfra "mycourse-io-be/internal/media/infra" //nolint:depguard // delivery uses infra.ParseMetadataFromRaw utility; TODO: expose via application port
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/utils"
)

func toUploadMetadataDTO(meta domain.UploadFileMetadata) UploadFileMetadata {
	return UploadFileMetadata{
		SizeBytes: meta.SizeBytes, WidthBytes: meta.WidthBytes, HeightBytes: meta.HeightBytes,
		MimeType: meta.MimeType, Extension: meta.Extension, DurationSeconds: meta.DurationSeconds,
		Bitrate: meta.Bitrate, FPS: meta.FPS, VideoCodec: meta.VideoCodec, AudioCodec: meta.AudioCodec,
		HasAudio: meta.HasAudio, IsHDR: meta.IsHDR, PageCount: meta.PageCount,
		HasPassword: meta.HasPassword, ArchiveEntries: meta.ArchiveEntries,
	}
}

// ToUploadFileResponse maps a domain.File to the public API UploadFileResponse DTO.
func ToUploadFileResponse(file domain.File) UploadFileResponse {
	return toUploadFileResponse(file)
}

// ToUploadFileResponsePtr maps a *domain.File to *UploadFileResponse; returns nil if file is nil.
func ToUploadFileResponsePtr(file *domain.File) *UploadFileResponse {
	if file == nil {
		return nil
	}
	r := toUploadFileResponse(*file)
	return &r
}

func toUploadFileResponse(file domain.File) UploadFileResponse {
	return UploadFileResponse{
		ID: file.ID, Kind: file.Kind, Filename: file.Filename, MimeType: file.MimeType,
		SizeBytes: file.SizeBytes, Status: file.Status, B2BucketName: file.B2BucketName,
		URL: file.URL, ObjectKey: file.ObjectKey, BunnyVideoID: file.BunnyVideoID,
		BunnyLibraryID: file.BunnyLibraryID, VideoID: file.VideoID, ThumbnailURL: file.ThumbnailURL,
		EmbededHTML: file.EmbededHTML, Duration: file.Duration, VideoProvider: file.VideoProvider,
		Metadata: toUploadMetadataDTO(file.Metadata), RowVersion: file.RowVersion,
		ContentFingerprint: file.ContentFingerprint,
	}
}

func toUploadFileResponses(files []domain.File) []UploadFileResponse {
	out := make([]UploadFileResponse, 0, len(files))
	for _, f := range files {
		out = append(out, toUploadFileResponse(f))
	}
	return out
}

func toUploadFileResponsesFromPointers(files []*domain.File) []UploadFileResponse {
	out := make([]UploadFileResponse, 0, len(files))
	for _, p := range files {
		if p != nil {
			out = append(out, toUploadFileResponse(*p))
		}
	}
	return out
}

func toFilterDomain(q FileFilterRequest) domain.FileFilter {
	return domain.FileFilter{
		Page: q.getPage(), PageSize: q.getPerPage(),
		Provider: q.Provider, Kind: q.Kind,
	}
}

// BindCreateFileMultipart reads optional multipart text fields into CreateFileInput.
// Kind and metadata are intentionally ignored (server-owned fields).
func BindCreateFileMultipart(c *gin.Context) (application.CreateFileInput, error) {
	return bindCreateFileMultipart(c)
}

// bindCreateFileMultipart reads optional multipart text fields into CreateFileInput.
func bindCreateFileMultipart(c *gin.Context) (application.CreateFileInput, error) {
	if _, err := mediainfra.ParseMetadataFromRaw(c.PostForm("metadata")); err != nil {
		return application.CreateFileInput{}, err
	}
	return application.CreateFileInput{
		ObjectKey: c.PostForm("object_key"),
	}, nil
}

// bindUpdateFileMultipart reads optional multipart text fields into UpdateFileInput.
func bindUpdateFileMultipart(c *gin.Context) (application.UpdateFileInput, error) {
	if _, err := mediainfra.ParseMetadataFromRaw(c.PostForm("metadata")); err != nil {
		return application.UpdateFileInput{}, err
	}
	req := application.UpdateFileInput{
		ReuseMediaID:          strings.TrimSpace(c.PostForm("reuse_media_id")),
		SkipUploadIfUnchanged: utils.ParseBoolLoose(c.PostForm("skip_upload_if_unchanged")),
	}
	if ev := strings.TrimSpace(c.PostForm("expected_row_version")); ev != "" {
		v, perr := strconv.ParseInt(ev, 10, 64)
		if perr != nil {
			return application.UpdateFileInput{}, fmt.Errorf("%s", constants.MsgExpectedRowVersionInteger)
		}
		req.ExpectedRowVersion = &v
	}
	return req, nil
}

func bindUpdateAndCreateMultipart(c *gin.Context) (application.UpdateFileInput, application.CreateFileInput, error) {
	updateReq, err := bindUpdateFileMultipart(c)
	if err != nil {
		return application.UpdateFileInput{}, application.CreateFileInput{}, err
	}
	createReq, err := bindCreateFileMultipart(c)
	return updateReq, createReq, err
}

func unmarshalBunnyWebhookRequest(raw []byte) (BunnyVideoWebhookRequest, error) {
	// json.Unmarshal via stdlib
	var req BunnyVideoWebhookRequest
	if err := jsonUnmarshal(raw, &req); err != nil {
		return BunnyVideoWebhookRequest{}, fmt.Errorf("%w: %v", errBunnyWebhookJSONInvalid, err)
	}
	return req, nil
}

func validateBunnyWebhookRequest(req BunnyVideoWebhookRequest) error {
	if req.VideoLibraryID <= 0 || strings.TrimSpace(req.VideoGUID) == "" || req.Status < 0 || req.Status > 10 {
		return errBunnyWebhookPayloadInvalid
	}
	return nil
}
