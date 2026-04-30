package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Backblaze/blazer/b2"
	gcdn "github.com/G-Core/gcorelabscdn-go"
	"github.com/G-Core/gcorelabscdn-go/gcore/provider"
	bunnystorage "github.com/l0wl3vel/bunny-storage-go-sdk"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/helper"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/setting"
)

func NewCloudClientsFromEnv() (*entities.CloudClients, error) {
	out := &entities.CloudClients{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}

	keyID := strings.TrimSpace(os.Getenv("MEDIA_B2_KEY_ID"))
	appKey := strings.TrimSpace(os.Getenv("MEDIA_B2_APP_KEY"))
	bucket := strings.TrimSpace(os.Getenv("MEDIA_B2_BUCKET"))
	if keyID != "" && appKey != "" && bucket != "" {
		client, err := b2.NewClient(context.Background(), keyID, appKey)
		if err != nil {
			return nil, err
		}
		out.B2Client = client
		out.B2BucketName = bucket
	}

	gcoreAPIToken := strings.TrimSpace(os.Getenv("MEDIA_GCORE_API_TOKEN"))
	if gcoreAPIToken != "" {
		apiClient := provider.NewClient(
			utils.NormalizeBaseURL(os.Getenv("MEDIA_GCORE_API_BASE_URL"), "https://api.gcore.com"),
			provider.WithSignerFunc(func(req *http.Request) error {
				req.Header.Set("Authorization", "APIKey "+gcoreAPIToken)
				return nil
			}),
		)
		out.GcoreService = gcdn.NewService(apiClient)
	}

	bunnyEndpoint := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_STORAGE_ENDPOINT"))
	bunnyPassword := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_STORAGE_PASSWORD"))
	if bunnyEndpoint != "" && bunnyPassword != "" {
		parsed, err := url.Parse("https://" + strings.TrimLeft(bunnyEndpoint, "/"))
		if err != nil {
			return nil, err
		}
		client := bunnystorage.NewClient(*parsed, bunnyPassword)
		out.BunnyStorage = &client
	}
	return out, nil
}

// effectiveB2Bucket prefers YAML/media.b2_bucket after setting.Setup(); falls back to env bucket from constructor.
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
	token := helper.EncodeLocalObjectKey(secret, objectKey)
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
	origin := utils.NormalizeBaseURL(setting.MediaSetting.B2BaseURL, bucket.BaseURL())
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
	}, nil
}

func UploadBunnyVideo(c *entities.CloudClients, ctx context.Context, filename string, payload []byte, objectKey string, meta entities.RawMetadata) (entities.ProviderUploadResult, error) {
	_ = objectKey // Bunny object key is the API GUID after create.
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return entities.ProviderUploadResult{}, &pkgerrors.ProviderError{Code: errcode.BunnyStreamNotConfigured}
	}
	apiBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamAPIBase, "https://video.bunnycdn.com")

	createBody, _ := json.Marshal(map[string]string{"title": filename})
	createURL := fmt.Sprintf("%s/library/%s/videos", apiBase, libraryID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(createBody))
	if err != nil {
		return entities.ProviderUploadResult{}, err
	}
	req.Header.Set("AccessKey", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return entities.ProviderUploadResult{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return entities.ProviderUploadResult{}, &pkgerrors.ProviderError{
			Code: errcode.BunnyCreateFailed,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf("bunny create video: HTTP %d", resp.StatusCode),
		}
	}
	var created struct {
		GUID string `json:"guid"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return entities.ProviderUploadResult{}, &pkgerrors.ProviderError{
			Code: errcode.BunnyInvalidResponse,
			Msg:  err.Error(),
			Err:  err,
		}
	}
	if created.GUID == "" {
		return entities.ProviderUploadResult{}, &pkgerrors.ProviderError{
			Code: errcode.BunnyInvalidResponse,
			Msg:  "bunny stream did not return video guid",
			Err:  errors.New("missing guid"),
		}
	}

	uploadURL := fmt.Sprintf("%s/library/%s/videos/%s", apiBase, libraryID, created.GUID)
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(payload))
	if err != nil {
		return entities.ProviderUploadResult{}, err
	}
	uploadReq.Header.Set("AccessKey", apiKey)
	uploadReq.Header.Set("Content-Type", "application/octet-stream")
	uploadResp, err := c.HTTPClient.Do(uploadReq)
	if err != nil {
		return entities.ProviderUploadResult{}, err
	}
	defer uploadResp.Body.Close()
	uploadBody, _ := io.ReadAll(uploadResp.Body)
	if uploadResp.StatusCode >= 300 {
		return entities.ProviderUploadResult{}, &pkgerrors.ProviderError{
			Code: errcode.BunnyUploadFailed,
			Msg:  strings.TrimSpace(string(uploadBody)),
			Err:  fmt.Errorf("bunny upload video: HTTP %d", uploadResp.StatusCode),
		}
	}

	stream := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "https://iframe.mediadelivery.net/play")
	playURL := fmt.Sprintf("%s/%s/%s", stream, libraryID, created.GUID)
	if meta == nil {
		meta = entities.RawMetadata{}
	}
	meta["video_guid"] = created.GUID
	meta["bunny_video_id"] = created.GUID
	meta["bunny_library_id"] = libraryID
	meta["video_provider"] = "bunny_stream"
	return entities.ProviderUploadResult{
		URL:       playURL,
		OriginURL: playURL,
		ObjectKey: created.GUID,
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
		return fmt.Errorf("Bunny Stream is not configured")
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
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bunny delete video failed: %s", string(body))
	}
	return nil
}

func GetBunnyVideoByID(c *entities.CloudClients, ctx context.Context, videoGUID string) (*entities.BunnyVideoDetail, error) {
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	apiKey := strings.TrimSpace(setting.MediaSetting.BunnyStreamAPIKey)
	if libraryID == "" || apiKey == "" {
		return nil, &pkgerrors.ProviderError{Code: errcode.BunnyStreamNotConfigured}
	}
	apiBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamAPIBase, "https://video.bunnycdn.com")
	u := fmt.Sprintf("%s/library/%s/videos/%s", apiBase, libraryID, videoGUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("AccessKey", apiKey)
	req.Header.Set("accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, &pkgerrors.ProviderError{
			Code: errcode.BunnyVideoNotFound,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf("bunny get video: HTTP %d", resp.StatusCode),
		}
	}
	if resp.StatusCode >= 300 {
		return nil, &pkgerrors.ProviderError{
			Code: errcode.BunnyGetVideoFailed,
			Msg:  strings.TrimSpace(string(body)),
			Err:  fmt.Errorf("bunny get video: HTTP %d", resp.StatusCode),
		}
	}

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
			Err:  errors.New("missing guid"),
		}
	}
	return &out, nil
}

func BuildPublicURL(provider constants.FileProvider, objectKey string) string {
	switch provider {
	case constants.FileProviderLocal:
		secret := strings.TrimSpace(setting.MediaSetting.LocalFileURLSecret)
		if secret == "" {
			secret = "mycourse-local-file-secret"
		}
		return "/api/v1/media/files/local/" + helper.EncodeLocalObjectKey(secret, objectKey)
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
