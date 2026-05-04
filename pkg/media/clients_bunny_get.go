package media

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/setting"
)

func bunnyStreamAuthorizedGET(ctx context.Context, hc *http.Client, urlStr, apiKey string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("AccessKey", apiKey)
	req.Header.Set("accept", "application/json")
	return hc.Do(req)
}

func bunnyVideoGetError(resp *http.Response, body []byte) error {
	if resp.StatusCode == http.StatusNotFound {
		return &pkgerrors.ProviderError{
			Code: errcode.BunnyVideoNotFound,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf("bunny get video: HTTP %d", resp.StatusCode),
		}
	}
	return &pkgerrors.ProviderError{
		Code: errcode.BunnyGetVideoFailed,
		Msg:  strings.TrimSpace(string(body)),
		Err:  fmt.Errorf("bunny get video: HTTP %d", resp.StatusCode),
	}
}

func decodeBunnyVideoDetailBody(body []byte) (*entities.BunnyVideoDetail, error) {
	var out entities.BunnyVideoDetail
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, &pkgerrors.ProviderError{
			Code: errcode.BunnyInvalidResponse,
			Msg:  err.Error(),
			Err:  err,
		}
	}
	if strings.TrimSpace(out.GUID) == "" {
		return nil, &pkgerrors.ProviderError{
			Code: errcode.BunnyInvalidResponse,
			Msg:  "bunny stream did not return video guid",
			Err:  pkgerrors.ErrBunnyStreamResponseMissingGUID,
		}
	}
	EnrichBunnyVideoDetail(&out)
	return &out, nil
}

func parseBunnyVideoGetResponse(resp *http.Response) (*entities.BunnyVideoDetail, error) {
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, bunnyVideoGetError(resp, body)
	}
	return decodeBunnyVideoDetailBody(body)
}

// GetBunnyVideoByID fetches Bunny Stream video JSON for the given GUID (library from settings).
func GetBunnyVideoByID(c *entities.CloudClients, ctx context.Context, videoGUID string) (*entities.BunnyVideoDetail, error) {
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return nil, &pkgerrors.ProviderError{Code: errcode.BunnyStreamNotConfigured}
	}
	apiBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamAPIBase, "https://video.bunnycdn.com")
	u := fmt.Sprintf("%s/library/%s/videos/%s", apiBase, libraryID, videoGUID)
	resp, err := bunnyStreamAuthorizedGET(ctx, c.HTTPClient, u, apiKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return parseBunnyVideoGetResponse(resp)
}
