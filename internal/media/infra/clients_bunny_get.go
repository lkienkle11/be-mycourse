package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/logger"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"

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
		return &domain.ProviderError{
			Code: apperrors.BunnyVideoNotFound,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf(constants.MsgBunnyGetVideoHTTP, resp.StatusCode),
		}
	}
	return &domain.ProviderError{
		Code: apperrors.BunnyGetVideoFailed,
		Msg:  strings.TrimSpace(string(body)),
		Err:  fmt.Errorf(constants.MsgBunnyGetVideoHTTP, resp.StatusCode),
	}
}

func decodeBunnyVideoDetailBody(body []byte) (*domain.BunnyVideoDetail, error) {
	var out domain.BunnyVideoDetail
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, &domain.ProviderError{
			Code: apperrors.BunnyInvalidResponse,
			Msg:  err.Error(),
			Err:  err,
		}
	}
	if strings.TrimSpace(out.GUID) == "" {
		return nil, &domain.ProviderError{
			Code: apperrors.BunnyInvalidResponse,
			Msg:  constants.MsgBunnyStreamResponseMissingVideoGUID,
			Err:  apperrors.ErrBunnyStreamResponseMissingGUID,
		}
	}
	EnrichBunnyVideoDetail(&out)
	return &out, nil
}

func parseBunnyVideoGetResponse(ctx context.Context, resp *http.Response) (*domain.BunnyVideoDetail, error) {
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
func GetBunnyVideoByID(c *CloudClients, ctx context.Context, videoGUID string) (*domain.BunnyVideoDetail, error) {
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return nil, &domain.ProviderError{Code: apperrors.BunnyStreamNotConfigured}
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
