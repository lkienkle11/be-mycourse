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
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/setting"
)

// effectiveB2Bucket prefers setting.MediaSetting.B2Bucket after setting.Setup(); falls back to B2BucketName from NewCloudClientsFromSetting.
func effectiveB2Bucket(c *entities.CloudClients) string {
	if b := strings.TrimSpace(setting.MediaSetting.B2Bucket); b != "" {
		return b
	}
	return strings.TrimSpace(c.B2BucketName)
}

func UploadLocal(_ *entities.CloudClients, objectKey string, _ entities.RawMetadata) (entities.ProviderUploadResult, error) {
	secret := strings.TrimSpace(setting.MediaSetting.LocalFileURLSecret)
	if secret == "" {
		secret = "mycourse-local-file-secret"
	}
	token := EncodeLocalObjectKey(secret, objectKey)
	path := "/api/v1/media/files/local/" + token
	return entities.ProviderUploadResult{
		URL:       path,
		OriginURL: objectKey,
		ObjectKey: objectKey,
		Metadata:  entities.RawMetadata{},
	}, nil
}

func UploadB2(c *entities.CloudClients, ctx context.Context, objectKey string, file io.Reader, meta entities.RawMetadata) (entities.ProviderUploadResult, error) {
	if c.B2Client == nil {
		return entities.ProviderUploadResult{}, fmt.Errorf("B2 client is not configured")
	}
	bucketName := effectiveB2Bucket(c)
	if bucketName == "" {
		return entities.ProviderUploadResult{}, &pkgerrors.ProviderError{Code: errcode.B2BucketNotConfigured}
	}
	bucket, err := c.B2Client.Bucket(ctx, bucketName)
	if err != nil {
		return entities.ProviderUploadResult{}, err
	}
	key := strings.TrimLeft(objectKey, "/")
	obj := bucket.Object(key)
	writer := obj.NewWriter(ctx)
	if _, err := io.Copy(writer, file); err != nil {
		return entities.ProviderUploadResult{}, err
	}
	if err := writer.Close(); err != nil {
		return entities.ProviderUploadResult{}, err
	}
	return b2UploadResultURLs(bucketName, key, bucket.BaseURL(), meta), nil
}

func b2UploadResultURLs(bucketName, key, bucketBaseURL string, meta entities.RawMetadata) entities.ProviderUploadResult {
	origin := utils.NormalizeBaseURL(setting.MediaSetting.B2BaseURL, bucketBaseURL)
	cdn := utils.NormalizeBaseURL(setting.MediaSetting.GcoreCDNURL, origin)
	publicURL := utils.JoinURLPathSegments(cdn, bucketName, key)
	if meta == nil {
		meta = entities.RawMetadata{}
	}
	meta["b2_bucket_name"] = bucketName
	return entities.ProviderUploadResult{
		URL:       publicURL,
		OriginURL: utils.JoinURLPathSegments(origin, key),
		ObjectKey: key,
		Metadata:  meta,
	}
}

func bunnyCreateStreamVideo(c *entities.CloudClients, ctx context.Context, apiBase, libraryID, apiKey, filename string) (string, error) {
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
		return "", &pkgerrors.ProviderError{
			Code: errcode.BunnyCreateFailed,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf("bunny create video: HTTP %d", resp.StatusCode),
		}
	}
	return decodeBunnyCreateVideoGUID(body)
}

func decodeBunnyCreateVideoGUID(body []byte) (string, error) {
	var created struct {
		GUID string `json:"guid"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return "", &pkgerrors.ProviderError{
			Code: errcode.BunnyInvalidResponse,
			Msg:  err.Error(),
			Err:  err,
		}
	}
	if created.GUID == "" {
		return "", &pkgerrors.ProviderError{
			Code: errcode.BunnyInvalidResponse,
			Msg:  "bunny stream did not return video guid",
			Err:  pkgerrors.ErrBunnyStreamResponseMissingGUID,
		}
	}
	return created.GUID, nil
}

func bunnyPutStreamVideoPayload(c *entities.CloudClients, ctx context.Context, apiBase, libraryID, apiKey, videoGUID string, payload []byte) error {
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
		return &pkgerrors.ProviderError{
			Code: errcode.BunnyUploadFailed,
			Msg:  strings.TrimSpace(string(uploadBody)),
			Err:  fmt.Errorf("bunny upload video: HTTP %d", uploadResp.StatusCode),
		}
	}
	return nil
}

func bunnyUploadApplyMetadata(meta entities.RawMetadata, c *entities.CloudClients, ctx context.Context, guid, libraryID, stream string) {
	meta["video_guid"] = guid
	meta["bunny_video_id"] = guid
	meta["bunny_library_id"] = libraryID
	meta["video_provider"] = "bunny_stream"
	if detail, derr := GetBunnyVideoByID(c, ctx, guid); derr == nil {
		ApplyBunnyDetailToMetadata(meta, detail, libraryID, stream)
	}
}

func UploadBunnyVideo(c *entities.CloudClients, ctx context.Context, filename string, payload []byte, objectKey string, meta entities.RawMetadata) (entities.ProviderUploadResult, error) {
	_ = objectKey // Bunny object key is the API GUID after create.
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return entities.ProviderUploadResult{}, &pkgerrors.ProviderError{Code: errcode.BunnyStreamNotConfigured}
	}
	apiBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamAPIBase, "https://video.bunnycdn.com")

	guid, err := bunnyCreateStreamVideo(c, ctx, apiBase, libraryID, apiKey, filename)
	if err != nil {
		return entities.ProviderUploadResult{}, err
	}
	if err := bunnyPutStreamVideoPayload(c, ctx, apiBase, libraryID, apiKey, guid, payload); err != nil {
		return entities.ProviderUploadResult{}, err
	}
	stream := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "https://iframe.mediadelivery.net/play")
	playURL := fmt.Sprintf("%s/%s/%s", stream, libraryID, guid)
	if meta == nil {
		meta = entities.RawMetadata{}
	}
	bunnyUploadApplyMetadata(meta, c, ctx, guid, libraryID, stream)
	return entities.ProviderUploadResult{
		URL:       playURL,
		OriginURL: playURL,
		ObjectKey: guid,
		Metadata:  meta,
	}, nil
}

func DeleteB2Object(c *entities.CloudClients, ctx context.Context, objectKey string) error {
	if c.B2Client == nil {
		return fmt.Errorf("B2 client is not configured")
	}
	bucketName := effectiveB2Bucket(c)
	if bucketName == "" {
		return &pkgerrors.ProviderError{Code: errcode.B2BucketNotConfigured}
	}
	bucket, err := c.B2Client.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}
	return bucket.Object(strings.TrimLeft(objectKey, "/")).Delete(ctx)
}

func DeleteBunnyVideo(c *entities.CloudClients, ctx context.Context, videoGUID string) error {
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return fmt.Errorf("bunny stream is not configured")
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
		return fmt.Errorf("bunny delete video failed: %s", string(body))
	}
	return nil
}

func BuildPublicURL(provider constants.FileProvider, objectKey string) string {
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
		cdn := utils.NormalizeBaseURL(setting.MediaSetting.GcoreCDNURL, "https://cdn.mycourse.local")
		key := strings.TrimLeft(objectKey, "/")
		if b := strings.TrimSpace(setting.MediaSetting.B2Bucket); b != "" {
			return utils.JoinURLPathSegments(cdn, b, key)
		}
		return utils.JoinURLPathSegments(cdn, key)
	}
}
