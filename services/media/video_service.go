package media

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/logic/helper"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
)

func GetVideoStatus(ctx context.Context, videoGUID string) (*dto.VideoStatusResponse, error) {
	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
		return nil, err
	}
	guid := strings.TrimSpace(videoGUID)
	if guid == "" {
		return nil, fmt.Errorf("video guid is required")
	}
	video, err := pkgmedia.Cloud.GetBunnyVideoByID(ctx, guid)
	if err != nil {
		return nil, err
	}
	return &dto.VideoStatusResponse{
		Status: utils.BunnyVideoStatus(video.Status).StatusString(),
	}, nil
}

func HandleBunnyVideoWebhook(ctx context.Context, req dto.BunnyVideoWebhookRequest) error {
	if err := helper.RequireInitialized(pkgmedia.Cloud); err != nil {
		return err
	}
	if req.Status != utils.FinishedWebhookBunnyStatus {
		return nil
	}

	video, err := pkgmedia.Cloud.GetBunnyVideoByID(ctx, strings.TrimSpace(req.VideoGUID))
	if err != nil {
		return err
	}

	re := regexp.MustCompile(utils.SignBunnyIFrameRegex)
	playbackURL := pkgmedia.BuildPublicURL(constants.FileProviderBunny, strings.TrimSpace(req.VideoGUID))
	_ = re.ReplaceAllString(playbackURL, "")
	_ = int64(video.Length)

	// TODO: persist duration xuống bảng files/lessons khi DB ready.
	return nil
}
