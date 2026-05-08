package media

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/repository"
)

func GetVideoStatus(ctx context.Context, videoGUID string) (*dto.VideoStatusResponse, error) {
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return nil, err
	}
	guid := strings.TrimSpace(videoGUID)
	if guid == "" {
		return nil, pkgerrors.ErrMediaVideoGUIDRequired
	}
	video, err := pkgmedia.GetBunnyVideoByID(pkgmedia.Cloud, ctx, guid)
	if err != nil {
		return nil, err
	}
	return &dto.VideoStatusResponse{
		Status: pkgmedia.BunnyStatusString(video.Status),
	}, nil
}

// patchBunnyWebhookMetadataJSON merges stream telemetry into row.MetadataJSON and returns the decoded map for derived fields.
func patchBunnyWebhookMetadataJSON(row *models.MediaFile, video *entities.BunnyVideoDetail) (map[string]any, error) {
	raw := map[string]any{}
	_ = json.Unmarshal(row.MetadataJSON, &raw)
	raw["length"] = video.Length
	raw["width"] = video.Width
	raw["height"] = video.Height
	raw["framerate"] = video.Framerate
	raw["bitrate"] = video.Bitrate
	raw["video_codec"] = video.VideoCodec
	raw["audio_codec"] = video.AudioCodec
	streamBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "https://iframe.mediadelivery.net/play")
	libID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	pkgmedia.ApplyBunnyDetailToMetadata(raw, video, libID, streamBase)
	blob, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	row.MetadataJSON = blob
	return raw, nil
}

// applyBunnyFinishedWebhookToRow merges Bunny get-video fields into the media row and metadata JSON.
func applyBunnyFinishedWebhookToRow(row *models.MediaFile, video *entities.BunnyVideoDetail, videoGUID string) error {
	raw, err := patchBunnyWebhookMetadataJSON(row, video)
	if err != nil {
		return err
	}
	row.VideoID = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyVideoID]))
	if row.VideoID == "" {
		row.VideoID = pkgmedia.FormatBunnyVideoIDString(video)
	}
	row.ThumbnailURL = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyThumbnailURL]))
	row.EmbededHTML = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyEmbededHTML]))
	row.Duration = int64(video.Length)
	row.Status = constants.FileStatusReady
	guid := strings.TrimSpace(videoGUID)
	if row.URL == "" {
		row.URL = pkgmedia.BuildPublicURL(constants.FileProviderBunny, guid)
	}
	if row.OriginURL == "" {
		row.OriginURL = row.URL
	}
	if row.VideoProvider == "" {
		row.VideoProvider = "bunny_stream"
	}
	return nil
}

func HandleBunnyVideoWebhook(ctx context.Context, req dto.BunnyVideoWebhookRequest) error {
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return err
	}
	status := req.Status
	if !isBunnyWebhookStatusSupported(status) {
		return nil
	}
	if !isBunnyWebhookFinishStatus(status) {
		if markBunnyWebhookFailedStatus(req.VideoGUID, status) != nil {
			return nil
		}
		return nil
	}
	return applyBunnyWebhookFinishedStatus(ctx, req.VideoGUID)
}

func applyBunnyWebhookFinishedStatus(ctx context.Context, videoGUID string) error {
	trimmedGUID := strings.TrimSpace(videoGUID)
	video, err := pkgmedia.GetBunnyVideoByID(pkgmedia.Cloud, ctx, trimmedGUID)
	if err != nil {
		return err
	}
	repo := repository.New(models.DB).Media
	row, err := repo.GetByBunnyVideoID(trimmedGUID)
	if err != nil {
		// idempotent retry safety: if local DB row does not exist, acknowledge without failing webhook.
		return nil
	}
	if err := applyBunnyFinishedWebhookToRow(row, video, trimmedGUID); err != nil {
		return err
	}
	return repo.UpsertByObjectKey(row)
}

func markBunnyWebhookFailedStatus(videoGUID string, status int) error {
	if status != constants.BunnyFailed && status != constants.BunnyPresignedUploadFailed {
		return nil
	}
	repo := repository.New(models.DB).Media
	row, err := repo.GetByBunnyVideoID(strings.TrimSpace(videoGUID))
	if err != nil {
		return err
	}
	row.Status = constants.FileStatusFailed
	return repo.UpsertByObjectKey(row)
}

func isBunnyWebhookFinishStatus(status int) bool {
	return status == constants.BunnyFinished || status == constants.BunnyResolutionFinished
}

func isBunnyWebhookStatusSupported(status int) bool {
	switch status {
	case constants.BunnyQueued,
		constants.BunnyProcessing,
		constants.BunnyEncoding,
		constants.BunnyFinished,
		constants.BunnyResolutionFinished,
		constants.BunnyFailed,
		constants.BunnyPresignedUploadStarted,
		constants.BunnyPresignedUploadFinished,
		constants.BunnyPresignedUploadFailed,
		constants.BunnyCaptionsGenerated,
		constants.BunnyTitleOrDescriptionGenerated:
		return true
	default:
		return false
	}
}
