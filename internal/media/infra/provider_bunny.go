package infra

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
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

func bunnyCreateStreamVideo(c *CloudClients, ctx context.Context, apiBase, libraryID, apiKey, filename string) (string, error) {
	createBody, _ := json.Marshal(map[string]string{"title": filename})
	createURL := fmt.Sprintf("%s/library/%s/videos", apiBase, libraryID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(createBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("AccessKey", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", &domain.ProviderError{
			Code: apperrors.BunnyCreateFailed,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf(constants.MsgBunnyCreateVideoHTTP, resp.StatusCode),
		}
	}
	return decodeBunnyCreateVideoGUID(body)
}

func decodeBunnyCreateVideoGUID(body []byte) (string, error) {
	var created struct {
		GUID string `json:"guid"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return "", &domain.ProviderError{
			Code: apperrors.BunnyInvalidResponse,
			Msg:  err.Error(),
			Err:  err,
		}
	}
	if created.GUID == "" {
		return "", &domain.ProviderError{
			Code: apperrors.BunnyInvalidResponse,
			Msg:  constants.MsgBunnyStreamResponseMissingVideoGUID,
			Err:  apperrors.ErrBunnyStreamResponseMissingGUID,
		}
	}
	return created.GUID, nil
}

func bunnyPutStreamVideoPayload(c *CloudClients, ctx context.Context, apiBase, libraryID, apiKey, videoGUID string, payload []byte) error {
	uploadURL := fmt.Sprintf("%s/library/%s/videos/%s", apiBase, libraryID, videoGUID)
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	uploadReq.Header.Set("AccessKey", apiKey)
	uploadReq.Header.Set("Content-Type", "application/octet-stream")
	uploadResp, err := c.HTTPClient.Do(uploadReq)
	if err != nil {
		return err
	}
	defer func() { _ = uploadResp.Body.Close() }()
	uploadBody, _ := io.ReadAll(uploadResp.Body)
	if uploadResp.StatusCode >= 300 {
		return &domain.ProviderError{
			Code: apperrors.BunnyUploadFailed,
			Msg:  strings.TrimSpace(string(uploadBody)),
			Err:  fmt.Errorf(constants.MsgBunnyUploadVideoHTTP, uploadResp.StatusCode),
		}
	}
	return nil
}

func bunnyUploadApplyMetadata(meta domain.RawMetadata, c *CloudClients, ctx context.Context, guid, libraryID, stream string) {
	meta["video_guid"] = guid
	meta["bunny_video_id"] = guid
	meta["bunny_library_id"] = libraryID
	meta["video_provider"] = "bunny_stream"
	if detail, derr := GetBunnyVideoByID(c, ctx, guid); derr == nil {
		ApplyBunnyDetailToMetadata(meta, detail, libraryID, stream)
	}
}

func UploadBunnyVideo(c *CloudClients, ctx context.Context, filename string, payload []byte, objectKey string, meta domain.RawMetadata) (domain.ProviderUploadResult, error) {
	_ = objectKey // Bunny object key is the API GUID after create.
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return domain.ProviderUploadResult{}, &domain.ProviderError{Code: apperrors.BunnyStreamNotConfigured}
	}
	apiBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamAPIBase, "https://video.bunnycdn.com")

	guid, err := bunnyCreateStreamVideo(c, ctx, apiBase, libraryID, apiKey, filename)
	if err != nil {
		return domain.ProviderUploadResult{}, err
	}
	if err := bunnyPutStreamVideoPayload(c, ctx, apiBase, libraryID, apiKey, guid, payload); err != nil {
		return domain.ProviderUploadResult{}, err
	}
	stream := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "https://iframe.mediadelivery.net/play")
	playURL := fmt.Sprintf("%s/%s/%s", stream, libraryID, guid)
	if meta == nil {
		meta = domain.RawMetadata{}
	}
	bunnyUploadApplyMetadata(meta, c, ctx, guid, libraryID, stream)
	return domain.ProviderUploadResult{
		URL:       playURL,
		OriginURL: playURL,
		ObjectKey: guid,
		Metadata:  meta,
	}, nil
}

func DeleteBunnyVideo(c *CloudClients, ctx context.Context, videoGUID string) error {
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return fmt.Errorf(constants.MsgBunnyStreamNotConfiguredRaw)
	}
	apiBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamAPIBase, "https://video.bunnycdn.com")
	u := fmt.Sprintf("%s/library/%s/videos/%s", apiBase, libraryID, videoGUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("AccessKey", apiKey)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(constants.MsgBunnyDeleteVideoFailed, string(body))
	}
	return nil
}

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

var bunnyStatusNames = map[int]string{
	domain.BunnyQueued:                      "queued",
	domain.BunnyProcessing:                  "processing",
	domain.BunnyEncoding:                    "encoding",
	domain.BunnyFinished:                    "finished",
	domain.BunnyResolutionFinished:          "resolution_finished",
	domain.BunnyFailed:                      "failed",
	domain.BunnyPresignedUploadStarted:      "presigned_upload_started",
	domain.BunnyPresignedUploadFinished:     "presigned_upload_finished",
	domain.BunnyPresignedUploadFailed:       "presigned_upload_failed",
	domain.BunnyCaptionsGenerated:           "captions_generated",
	domain.BunnyTitleOrDescriptionGenerated: "title_or_description_generated",
}

func BunnyStatusString(status int) string {
	if name, ok := bunnyStatusNames[status]; ok {
		return name
	}
	return "unknown"
}

// BunnyWebhookSigningSecret returns the signing secret for Bunny webhook validation.
// Source of truth is MediaSetting: read-only key first, then API key fallback for backward compatibility.
func BunnyWebhookSigningSecret() string {
	if key := strings.TrimSpace(setting.MediaSetting.BunnyStreamReadOnlyAPIKey); key != "" {
		return key
	}
	return strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
}

func BunnyWebhookSignatureExpectedHex(rawBody []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(rawBody)
	return hex.EncodeToString(mac.Sum(nil))
}

func IsBunnyWebhookSignatureValid(rawBody []byte, signature, version, algorithm, secret string) bool {
	if strings.TrimSpace(version) != domain.BunnyWebhookSignatureVersionV1 {
		return false
	}
	if strings.TrimSpace(strings.ToLower(algorithm)) != domain.BunnyWebhookSignatureAlgorithmHMAC {
		return false
	}
	received := strings.TrimSpace(strings.ToLower(signature))
	if len(received) != 64 {
		return false
	}
	for _, c := range received {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	expected := BunnyWebhookSignatureExpectedHex(rawBody, secret)
	if len(expected) != len(received) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(received)) == 1
}
