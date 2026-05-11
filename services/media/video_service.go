package media

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logger"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/repository"
)

func GetVideoStatus(ctx context.Context, videoGUID string) (*entities.VideoProviderStatus, error) {
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
	return &entities.VideoProviderStatus{
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

func bunnyWebhookHandleLogger(ctx context.Context, req dto.BunnyVideoWebhookRequest) *zap.Logger {
	return logger.FromContext(ctx).With(
		zap.String("component", "bunny_webhook"),
		zap.String("bunny_webhook_service", "HandleBunnyVideoWebhook"),
		zap.Int("video_library_id", req.VideoLibraryID),
		zap.String("video_guid", req.VideoGUID),
		zap.Int("status", req.Status),
	)
}

func handleBunnyWebhookNonFinishBranch(ctx context.Context, log *zap.Logger, videoGUID string, status int) {
	log.Debug("bunny webhook: non-finish status branch",
		zap.String("bunny_webhook_stage", "service_status_non_finish"),
	)
	if err := markBunnyWebhookFailedStatus(ctx, videoGUID, status); err != nil {
		log.Warn("bunny webhook: mark failed status returned error (acknowledged to provider anyway)",
			zap.String("bunny_webhook_stage", "service_mark_failed_error"),
			zap.Error(err),
		)
	}
}

func applyBunnyWebhookFinishedPersist(log *zap.Logger, row *models.MediaFile, video *entities.BunnyVideoDetail, trimmedGUID string) error {
	if err := applyBunnyFinishedWebhookToRow(row, video, trimmedGUID); err != nil {
		log.Warn("bunny webhook: applyBunnyFinishedWebhookToRow failed", zap.Error(err))
		return err
	}
	repo := repository.New(models.DB).Media
	if err := repo.UpsertByObjectKey(row); err != nil {
		log.Warn("bunny webhook: UpsertByObjectKey failed", zap.Error(err))
		return err
	}
	log.Debug("bunny webhook: DB row updated after finished webhook",
		zap.String("bunny_webhook_stage", "service_db_upsert_ok"),
		zap.String("media_file_id", row.ID),
	)
	return nil
}

func markBunnyWebhookFailedPersist(log *zap.Logger, videoGUID string) error {
	repo := repository.New(models.DB).Media
	row, err := repo.GetByBunnyVideoID(strings.TrimSpace(videoGUID))
	if err != nil {
		log.Debug("bunny webhook: mark failed — no DB row",
			zap.String("bunny_webhook_stage", "service_mark_failed_no_row"),
			zap.Error(err),
		)
		return err
	}
	row.Status = constants.FileStatusFailed
	if err := repo.UpsertByObjectKey(row); err != nil {
		log.Warn("bunny webhook: mark failed — UpsertByObjectKey error",
			zap.String("bunny_webhook_stage", "service_mark_failed_upsert_error"),
			zap.Error(err),
		)
		return err
	}
	log.Debug("bunny webhook: media row marked as failed",
		zap.String("bunny_webhook_stage", "service_mark_failed_ok"),
		zap.String("media_file_id", row.ID),
	)
	return nil
}

func HandleBunnyVideoWebhook(ctx context.Context, req dto.BunnyVideoWebhookRequest) error {
	log := bunnyWebhookHandleLogger(ctx, req)
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		log.Warn("bunny webhook: media cloud client not initialized", zap.Error(err))
		return err
	}
	status := req.Status
	if !isBunnyWebhookStatusSupported(status) {
		log.Debug("bunny webhook: unsupported status, no-op",
			zap.String("bunny_webhook_stage", "service_status_unsupported"),
		)
		return nil
	}
	if !isBunnyWebhookFinishStatus(status) {
		handleBunnyWebhookNonFinishBranch(ctx, log, req.VideoGUID, status)
		return nil
	}
	log.Debug("bunny webhook: finish status, applying to DB",
		zap.String("bunny_webhook_stage", "service_apply_finished_start"),
	)
	return applyBunnyWebhookFinishedStatus(ctx, req.VideoGUID)
}

func applyBunnyWebhookFinishedStatus(ctx context.Context, videoGUID string) error {
	log := logger.FromContext(ctx).With(
		zap.String("component", "bunny_webhook"),
		zap.String("bunny_webhook_service", "applyBunnyWebhookFinishedStatus"),
		zap.String("video_guid", videoGUID),
	)
	trimmedGUID := strings.TrimSpace(videoGUID)
	video, err := pkgmedia.GetBunnyVideoByID(pkgmedia.Cloud, ctx, trimmedGUID)
	if err != nil {
		log.Warn("bunny webhook: GetBunnyVideoByID failed", zap.Error(err))
		return err
	}
	log.Debug("bunny webhook: fetched video detail from Bunny API",
		zap.String("bunny_webhook_stage", "service_bunny_get_ok"),
	)
	repo := repository.New(models.DB).Media
	row, err := repo.GetByBunnyVideoID(trimmedGUID)
	if err != nil {
		// idempotent retry safety: if local DB row does not exist, acknowledge without failing webhook.
		log.Debug("bunny webhook: no local media row for video_guid, skipping DB update",
			zap.String("bunny_webhook_stage", "service_db_row_missing"),
			zap.Error(err),
		)
		return nil
	}
	return applyBunnyWebhookFinishedPersist(log, row, video, trimmedGUID)
}

func markBunnyWebhookFailedStatus(ctx context.Context, videoGUID string, status int) error {
	log := logger.FromContext(ctx).With(
		zap.String("component", "bunny_webhook"),
		zap.String("bunny_webhook_service", "markBunnyWebhookFailedStatus"),
		zap.String("video_guid", videoGUID),
		zap.Int("status", status),
	)
	if status != constants.BunnyFailed && status != constants.BunnyPresignedUploadFailed {
		log.Debug("bunny webhook: status is not a failure code, skipping mark",
			zap.String("bunny_webhook_stage", "service_mark_failed_skip"),
		)
		return nil
	}
	return markBunnyWebhookFailedPersist(log, videoGUID)
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
