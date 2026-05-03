package media

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/logic/helper"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/repository"
)

func GetVideoStatus(ctx context.Context, videoGUID string) (*dto.VideoStatusResponse, error) {
	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
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
		Status: helper.BunnyVideoStatus(video.Status).StatusString(),
	}, nil
}

func HandleBunnyVideoWebhook(ctx context.Context, req dto.BunnyVideoWebhookRequest) error {
	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
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
	helper.ApplyBunnyDetailToMetadata(raw, video, libID, streamBase)
	row.VideoID = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyVideoID]))
	if row.VideoID == "" {
		row.VideoID = helper.FormatBunnyVideoIDString(video)
	}
	row.ThumbnailURL = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyThumbnailURL]))
	row.EmbededHTML = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyEmbededHTML]))
	blob, _ := json.Marshal(raw)
	row.MetadataJSON = blob
	row.Duration = int64(video.Length)
	row.Status = constants.FileStatusReady
	if row.URL == "" {
		row.URL = pkgmedia.BuildPublicURL(constants.FileProviderBunny, strings.TrimSpace(req.VideoGUID))
	}
	if row.OriginURL == "" {
		row.OriginURL = row.URL
	}
	if row.VideoProvider == "" {
		row.VideoProvider = "bunny_stream"
	}
	return repo.UpsertByObjectKey(row)
}
