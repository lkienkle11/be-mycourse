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
		return nil, fmt.Errorf("video guid is required")
	}
	video, err := pkgmedia.GetBunnyVideoByID(pkgmedia.Cloud, ctx, guid)
	if err != nil {
		return nil, err
	}
	return &dto.VideoStatusResponse{
		Status: constants.BunnyVideoStatus(video.Status).StatusString(),
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
	if req.Status != constants.FinishedWebhookBunnyStatus {
		return nil
	}

	video, err := pkgmedia.GetBunnyVideoByID(pkgmedia.Cloud, ctx, strings.TrimSpace(req.VideoGUID))
	if err != nil {
		return err
	}
	repo := repository.New(models.DB).Media
	row, err := repo.GetByBunnyVideoID(strings.TrimSpace(req.VideoGUID))
	if err != nil {
		// idempotent retry safety: if local DB row does not exist, acknowledge without failing webhook.
		return nil
	}
	if err := applyBunnyFinishedWebhookToRow(row, video, req.VideoGUID); err != nil {
		return err
	}
	return repo.UpsertByObjectKey(row)
}
