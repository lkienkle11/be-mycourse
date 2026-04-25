package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Backblaze/blazer/b2"
	gcdn "github.com/G-Core/gcorelabscdn-go"
	"github.com/G-Core/gcorelabscdn-go/gcore/provider"
	bunnystorage "github.com/l0wl3vel/bunny-storage-go-sdk"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/helper"
	"mycourse-io-be/pkg/logic/utils"
)

type CloudClients struct {
	b2Client     *b2.Client
	b2BucketName string
	bunnyStorage *bunnystorage.Client
	gcoreService *gcdn.Service
	httpClient   *http.Client
}

func NewCloudClientsFromEnv() (*CloudClients, error) {
	out := &CloudClients{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	keyID := strings.TrimSpace(os.Getenv("MEDIA_B2_KEY_ID"))
	appKey := strings.TrimSpace(os.Getenv("MEDIA_B2_APP_KEY"))
	bucket := strings.TrimSpace(os.Getenv("MEDIA_B2_BUCKET"))
	if keyID != "" && appKey != "" && bucket != "" {
		client, err := b2.NewClient(context.Background(), keyID, appKey)
		if err != nil {
			return nil, err
		}
		out.b2Client = client
		out.b2BucketName = bucket
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
		out.gcoreService = gcdn.NewService(apiClient)
	}

	bunnyEndpoint := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_STORAGE_ENDPOINT"))
	bunnyPassword := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_STORAGE_PASSWORD"))
	if bunnyEndpoint != "" && bunnyPassword != "" {
		parsed, err := url.Parse("https://" + strings.TrimLeft(bunnyEndpoint, "/"))
		if err != nil {
			return nil, err
		}
		client := bunnystorage.NewClient(*parsed, bunnyPassword)
		out.bunnyStorage = &client
	}
	return out, nil
}

func (c *CloudClients) UploadLocal(objectKey string, meta entities.RawMetadata) (dto.UploadFileResponse, error) {
	secret := strings.TrimSpace(os.Getenv("LOCAL_FILE_URL_SECRET"))
	if secret == "" {
		secret = "mycourse-local-file-secret"
	}
	token := helper.EncodeLocalObjectKey(secret, objectKey)
	path := "/api/v1/media/files/local/" + token
	return dto.UploadFileResponse{
		URL:       path,
		OriginURL: objectKey,
		ObjectKey: objectKey,
		Provider:  string(constants.FileProviderLocal),
		Metadata:  meta,
	}, nil
}

func (c *CloudClients) UploadB2(ctx context.Context, objectKey string, file io.Reader, meta entities.RawMetadata) (dto.UploadFileResponse, error) {
	if c.b2Client == nil || c.b2BucketName == "" {
		return dto.UploadFileResponse{}, fmt.Errorf("B2 client is not configured")
	}
	bucket, err := c.b2Client.Bucket(ctx, c.b2BucketName)
	if err != nil {
		return dto.UploadFileResponse{}, err
	}
	key := strings.TrimLeft(objectKey, "/")
	obj := bucket.Object(key)
	writer := obj.NewWriter(ctx)
	if _, err := io.Copy(writer, file); err != nil {
		return dto.UploadFileResponse{}, err
	}
	if err := writer.Close(); err != nil {
		return dto.UploadFileResponse{}, err
	}
	origin := utils.NormalizeBaseURL(os.Getenv("MEDIA_B2_BASE_URL"), bucket.BaseURL())
	cdn := utils.NormalizeBaseURL(os.Getenv("MEDIA_GCORE_CDN_URL"), origin)
	return dto.UploadFileResponse{
		URL:       cdn + "/" + key,
		OriginURL: origin + "/" + key,
		ObjectKey: key,
		Provider:  string(constants.FileProviderB2),
		Metadata:  meta,
	}, nil
}

func (c *CloudClients) UploadBunnyVideo(ctx context.Context, filename string, payload []byte, objectKey string, meta entities.RawMetadata) (dto.UploadFileResponse, error) {
	libraryID := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_LIBRARY_ID"))
	apiKey := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_STREAM_API_KEY"))
	if libraryID == "" || apiKey == "" {
		return dto.UploadFileResponse{}, fmt.Errorf("Bunny Stream is not configured")
	}
	apiBase := utils.NormalizeBaseURL(os.Getenv("MEDIA_BUNNY_STREAM_API_BASE_URL"), "https://video.bunnycdn.com")

	createBody, _ := json.Marshal(map[string]string{"title": filename})
	createURL := fmt.Sprintf("%s/library/%s/videos", apiBase, libraryID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(createBody))
	if err != nil {
		return dto.UploadFileResponse{}, err
	}
	req.Header.Set("AccessKey", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return dto.UploadFileResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return dto.UploadFileResponse{}, fmt.Errorf("bunny create video failed: %s", string(body))
	}
	var created struct {
		GUID string `json:"guid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return dto.UploadFileResponse{}, err
	}
	if created.GUID == "" {
		return dto.UploadFileResponse{}, fmt.Errorf("bunny stream did not return video guid")
	}

	uploadURL := fmt.Sprintf("%s/library/%s/videos/%s", apiBase, libraryID, created.GUID)
	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(payload))
	if err != nil {
		return dto.UploadFileResponse{}, err
	}
	uploadReq.Header.Set("AccessKey", apiKey)
	uploadReq.Header.Set("Content-Type", "application/octet-stream")
	uploadResp, err := c.httpClient.Do(uploadReq)
	if err != nil {
		return dto.UploadFileResponse{}, err
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode >= 300 {
		body, _ := io.ReadAll(uploadResp.Body)
		return dto.UploadFileResponse{}, fmt.Errorf("bunny upload video failed: %s", string(body))
	}

	stream := utils.NormalizeBaseURL(os.Getenv("MEDIA_BUNNY_STREAM_BASE_URL"), "https://iframe.mediadelivery.net/play")
	playURL := fmt.Sprintf("%s/%s/%s", stream, libraryID, created.GUID)
	if meta == nil {
		meta = entities.RawMetadata{}
	}
	meta["video_guid"] = created.GUID
	meta["bunny_video_id"] = created.GUID
	meta["bunny_library_id"] = libraryID
	meta["video_provider"] = "bunny_stream"
	return dto.UploadFileResponse{
		URL:       playURL,
		OriginURL: playURL,
		ObjectKey: objectKey,
		Provider:  string(constants.FileProviderBunny),
		Metadata:  meta,
	}, nil
}

func (c *CloudClients) DeleteB2Object(ctx context.Context, objectKey string) error {
	if c.b2Client == nil || c.b2BucketName == "" {
		return fmt.Errorf("B2 client is not configured")
	}
	bucket, err := c.b2Client.Bucket(ctx, c.b2BucketName)
	if err != nil {
		return err
	}
	return bucket.Object(strings.TrimLeft(objectKey, "/")).Delete(ctx)
}

func (c *CloudClients) DeleteBunnyVideo(ctx context.Context, videoGUID string) error {
	libraryID := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_LIBRARY_ID"))
	apiKey := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_STREAM_API_KEY"))
	if libraryID == "" || apiKey == "" {
		return fmt.Errorf("Bunny Stream is not configured")
	}
	apiBase := utils.NormalizeBaseURL(os.Getenv("MEDIA_BUNNY_STREAM_API_BASE_URL"), "https://video.bunnycdn.com")
	u := fmt.Sprintf("%s/library/%s/videos/%s", apiBase, libraryID, videoGUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("AccessKey", apiKey)
	resp, err := c.httpClient.Do(req)
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

func BuildObjectKey(defaultKey, filename string) string {
	key := strings.TrimSpace(defaultKey)
	if key != "" {
		return strings.TrimLeft(key, "/")
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	base = strings.ReplaceAll(strings.TrimSpace(base), " ", "-")
	if base == "" {
		base = "file"
	}
	return fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), base, ext)
}

func BuildPublicURL(provider constants.FileProvider, objectKey string) string {
	switch provider {
	case constants.FileProviderLocal:
		secret := strings.TrimSpace(os.Getenv("LOCAL_FILE_URL_SECRET"))
		if secret == "" {
			secret = "mycourse-local-file-secret"
		}
		return "/api/v1/media/files/local/" + helper.EncodeLocalObjectKey(secret, objectKey)
	case constants.FileProviderBunny:
		base := utils.NormalizeBaseURL(os.Getenv("MEDIA_BUNNY_STREAM_BASE_URL"), "https://iframe.mediadelivery.net/play")
		libraryID := strings.TrimSpace(os.Getenv("MEDIA_BUNNY_LIBRARY_ID"))
		if libraryID == "" {
			libraryID = "00000"
		}
		return fmt.Sprintf("%s/%s/%s", base, libraryID, objectKey)
	default:
		cdn := utils.NormalizeBaseURL(os.Getenv("MEDIA_GCORE_CDN_URL"), "https://cdn.mycourse.local")
		return cdn + "/" + strings.TrimLeft(objectKey, "/")
	}
}
