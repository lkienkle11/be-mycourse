package application

import (
	"context"
	"math"
	"strings"
	"sync"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/logger"

	"go.uber.org/zap"
)

const missingVideoDurationBackfillLimit = 50

var pendingVideoDurationSync sync.Map

func videoDurationSecondsFromTelemetry(typedSeconds float64, bunnyLength float64) int64 {
	if typedSeconds > 0 {
		return int64(math.Floor(typedSeconds + 1e-9))
	}
	if bunnyLength > 0 {
		return int64(math.Floor(bunnyLength + 1e-9))
	}
	return 0
}

// scheduleVideoDurationRefresh enqueues async Bunny→DB sync for zero-duration videos.
// List/Get responses are not blocked; deduped by bunny_video_id.
func (s *MediaService) scheduleVideoDurationRefresh(ctx context.Context, files []domain.File) {
	for i := range files {
		if files[i].Kind != constants.FileKindVideo || files[i].Duration > 0 {
			continue
		}
		guid := strings.TrimSpace(files[i].BunnyVideoID)
		if guid == "" {
			continue
		}
		if _, loaded := pendingVideoDurationSync.LoadOrStore(guid, struct{}{}); loaded {
			continue
		}
		go func(videoGUID string) {
			defer pendingVideoDurationSync.Delete(videoGUID)
			if err := s.applyBunnyWebhookFinishedStatus(ctx, videoGUID); err != nil {
				logger.FromContext(ctx).Debug(
					"async video duration sync skipped",
					zap.String("video_guid", videoGUID),
					zap.Error(err),
				)
			}
		}(guid)
	}
}

// BackfillMissingVideoDurations pulls Bunny telemetry for READY videos whose
// media_files.duration is still 0 and persists length into the DB row.
func (s *MediaService) BackfillMissingVideoDurations(ctx context.Context) {
	log := logger.FromContext(ctx).With(zap.String("component", "media_duration_backfill"))
	guids, err := s.fileRepo.ListBunnyVideoGUIDsWithMissingDuration(ctx, missingVideoDurationBackfillLimit)
	if err != nil {
		log.Warn("list missing video durations failed", zap.Error(err))
		return
	}
	for _, guid := range guids {
		if err := s.applyBunnyWebhookFinishedStatus(ctx, guid); err != nil {
			log.Debug("video duration backfill skipped", zap.String("video_guid", guid), zap.Error(err))
		}
	}
}
