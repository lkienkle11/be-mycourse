package infra

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Backblaze/blazer/b2"
	gcdn "github.com/G-Core/gcorelabscdn-go"
	"github.com/G-Core/gcorelabscdn-go/gcore/provider"
	bunnystorage "github.com/l0wl3vel/bunny-storage-go-sdk"

	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

// NewCloudClientsFromSetting builds B2, Gcore API, and Bunny Storage SDK handles from
// setting.MediaSetting (YAML + .env resolved in setting.Setup). Call only after setting.Setup();
// main.go orders setting.Setup before pkg/media.Setup.
func attachB2FromSetting(out *CloudClients, ms *setting.Media) error {
	if ms == nil {
		return nil
	}
	keyID := strings.TrimSpace(ms.B2KeyID)
	appKey := strings.TrimSpace(ms.B2AppKey)
	bucket := strings.TrimSpace(ms.B2Bucket)
	if keyID == "" || appKey == "" || bucket == "" {
		return nil
	}
	client, err := b2.NewClient(context.Background(), keyID, appKey)
	if err != nil {
		return err
	}
	out.B2Client = client
	out.B2BucketName = bucket
	return nil
}

func attachGcoreFromSetting(out *CloudClients, ms *setting.Media) {
	if ms == nil {
		return
	}
	gcoreAPIToken := strings.TrimSpace(ms.GcoreAPIToken)
	if gcoreAPIToken == "" {
		return
	}
	apiClient := provider.NewClient(
		utils.NormalizeBaseURL(ms.GcoreAPIBaseURL, "https://api.gcore.com"),
		provider.WithSignerFunc(func(req *http.Request) error {
			req.Header.Set("Authorization", "APIKey "+gcoreAPIToken)
			return nil
		}),
	)
	out.GcoreService = gcdn.NewService(apiClient)
}

func attachBunnyStorageFromSetting(out *CloudClients, ms *setting.Media) error {
	if ms == nil {
		return nil
	}
	bunnyEndpoint := strings.TrimSpace(ms.BunnyStorageEndpoint)
	bunnyPassword := strings.TrimSpace(ms.BunnyStoragePassword)
	if bunnyEndpoint == "" || bunnyPassword == "" {
		return nil
	}
	parsed, err := url.Parse("https://" + strings.TrimLeft(bunnyEndpoint, "/"))
	if err != nil {
		return err
	}
	client := bunnystorage.NewClient(*parsed, bunnyPassword)
	out.BunnyStorage = &client
	return nil
}

func NewCloudClientsFromSetting() (*CloudClients, error) {
	out := &CloudClients{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
	ms := setting.MediaSetting
	if err := attachB2FromSetting(out, ms); err != nil {
		return nil, err
	}
	attachGcoreFromSetting(out, ms)
	if err := attachBunnyStorageFromSetting(out, ms); err != nil {
		return nil, err
	}
	return out, nil
}
