package delivery

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/media/application"
	"mycourse-io-be/internal/media/domain"
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
		EmbededHTML: file.EmbededHTML, DirectPlayURL: file.DirectPlayURL,
		HLSPlaylistURL: file.HLSPlaylistURL, PreviewAnimationURL: file.PreviewAnimationURL,
		Duration: file.Duration, VideoProvider: file.VideoProvider,
		Metadata: toUploadMetadataDTO(file.Metadata), RowVersion: file.RowVersion,
		ContentFingerprint: file.ContentFingerprint,
		CreatedAt:          file.CreatedAt, UpdatedAt: file.UpdatedAt,
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
	kind := q.Kind
	if q.Category != nil {
		switch *q.Category {
		case "video":
			videoKind := constants.FileKindVideo
			kind = &videoKind
		case "image", "document":
			fileKind := constants.FileKindFile
			kind = &fileKind
		}
	}
	sortBy := strings.TrimSpace(q.SortBy)
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := strings.TrimSpace(q.SortOrder)
	if sortOrder == "" {
		sortOrder = "desc"
	}
	return domain.FileFilter{
		Page: q.getPage(), PageSize: q.getPerPage(),
		Search:   strings.TrimSpace(q.Search),
		Provider: q.Provider, Kind: kind,
		SortBy: sortBy, SortOrder: sortOrder, Category: q.Category,
	}
}

// BindCreateFileMultipart reads optional multipart text fields into CreateFileInput.
// Kind and metadata are intentionally ignored (server-owned fields).
func BindCreateFileMultipart(c *gin.Context, gw domain.MediaGateway) (application.CreateFileInput, error) {
	return bindCreateFileMultipart(c, gw)
}

func validateMultipartMetadata(c *gin.Context, gw domain.MediaGateway) error {
	_, err := gw.ParseMetadataFromRaw(c.PostForm("metadata"))
	return err
}

// bindCreateFileMultipart reads optional multipart text fields into CreateFileInput.
func bindCreateFileMultipart(c *gin.Context, gw domain.MediaGateway) (application.CreateFileInput, error) {
	if err := validateMultipartMetadata(c, gw); err != nil {
		return application.CreateFileInput{}, err
	}
	return application.CreateFileInput{
		ObjectKey: c.PostForm("object_key"),
	}, nil
}

// bindUpdateFileMultipart reads optional multipart text fields into UpdateFileInput.
func bindUpdateFileMultipart(c *gin.Context, gw domain.MediaGateway) (application.UpdateFileInput, error) {
	if err := validateMultipartMetadata(c, gw); err != nil {
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

func bindUpdateAndCreateMultipart(c *gin.Context, gw domain.MediaGateway) (application.UpdateFileInput, application.CreateFileInput, error) {
	updateReq, err := bindUpdateFileMultipart(c, gw)
	if err != nil {
		return application.UpdateFileInput{}, application.CreateFileInput{}, err
	}
	createReq, err := bindCreateFileMultipart(c, gw)
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
