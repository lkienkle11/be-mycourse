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
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

func UploadLocal(_ *CloudClients, objectKey string, _ domain.RawMetadata) (domain.ProviderUploadResult, error) {
	secret := strings.TrimSpace(setting.MediaSetting.LocalFileURLSecret)
	if secret == "" {
		secret = "mycourse-local-file-secret"
	}
	token := EncodeLocalObjectKey(secret, objectKey)
	path := "/api/v1/media/files/local/" + token
	return domain.ProviderUploadResult{
		URL:       path,
		OriginURL: objectKey,
		ObjectKey: objectKey,
		Metadata:  domain.RawMetadata{},
	}, nil
}

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

func BuildPublicURL(provider string, objectKey string) string {
	switch provider {
	case constants.FileProviderLocal:
		secret := strings.TrimSpace(setting.MediaSetting.LocalFileURLSecret)
		if secret == "" {
			secret = "mycourse-local-file-secret"
		}
		return "/api/v1/media/files/local/" + EncodeLocalObjectKey(secret, objectKey)
	case constants.FileProviderBunny:
		base := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "https://iframe.mediadelivery.net/play")
		libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
		if libraryID == "" {
			libraryID = "00000"
		}
		return fmt.Sprintf("%s/%s/%s", base, libraryID, objectKey)
	default:
		return buildR2PublicURL(objectKey)
	}
}
