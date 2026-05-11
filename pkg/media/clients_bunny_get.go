package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logger"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/setting"

	"go.uber.org/zap"
)

// formatBunnyHTTPBodyForLog returns a human-readable body string for logs (pretty JSON when valid).
func formatBunnyHTTPBodyForLog(body []byte) string {
	if len(body) == 0 {
		return "(empty)"
	}
	if json.Valid(body) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, body, "", "  "); err == nil {
			return buf.String()
		}
	}
	return string(body)
}

func bunnyStreamAuthorizedGET(ctx context.Context, hc *http.Client, urlStr, apiKey string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("AccessKey", apiKey)
	req.Header.Set("accept", "application/json")

	// GET has no entity body; log explicitly so traces stay symmetric with response_body.
	requestBody := "(empty: HTTP GET has no request body)"
	logger.FromContext(ctx).Info(
		"bunny get video HTTP request",
		zap.String("method", req.Method),
		zap.String("url", urlStr),
		zap.String("request_body", requestBody),
		zap.String("header_accept", req.Header.Get("Accept")),
		zap.Bool("header_access_key_set", strings.TrimSpace(req.Header.Get("AccessKey")) != ""),
	)
	return hc.Do(req)
}

func bunnyVideoGetError(resp *http.Response, body []byte) error {
	if resp.StatusCode == http.StatusNotFound {
		return &pkgerrors.ProviderError{
			Code: errcode.BunnyVideoNotFound,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf(constants.MsgBunnyGetVideoHTTP, resp.StatusCode),
		}
	}
	return &pkgerrors.ProviderError{
		Code: errcode.BunnyGetVideoFailed,
		Msg:  strings.TrimSpace(string(body)),
		Err:  fmt.Errorf(constants.MsgBunnyGetVideoHTTP, resp.StatusCode),
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
			Msg:  constants.MsgBunnyStreamResponseMissingVideoGUID,
			Err:  pkgerrors.ErrBunnyStreamResponseMissingGUID,
		}
	}
	EnrichBunnyVideoDetail(&out)
	return &out, nil
}

func parseBunnyVideoGetResponse(ctx context.Context, resp *http.Response) (*entities.BunnyVideoDetail, error) {
	body, _ := io.ReadAll(resp.Body)
	logger.FromContext(ctx).Info(
		"bunny get video HTTP response",
		zap.Int("status", resp.StatusCode),
		zap.String("content_type", resp.Header.Get("Content-Type")),
		zap.Int64("content_length", resp.ContentLength),
		zap.String("response_body", formatBunnyHTTPBodyForLog(body)),
	)
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
	return parseBunnyVideoGetResponse(ctx, resp)
}
