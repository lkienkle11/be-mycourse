package infra

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	bunnystorage "github.com/l0wl3vel/bunny-storage-go-sdk"

	"mycourse-io-be/internal/shared/setting"
)

// NewCloudClientsFromSetting builds R2 (S3 API) and Bunny Storage SDK handles from
// setting.MediaSetting (YAML + .env resolved in setting.Setup). Call only after setting.Setup();
// main.go orders setting.Setup before pkg/media.Setup.
func attachR2FromSetting(out *CloudClients, ms *setting.Media) {
	if ms == nil {
		return
	}
	accessKey := strings.TrimSpace(ms.R2.AccessKeyID)
	secretKey := strings.TrimSpace(ms.R2.SecretAccessKey)
	bucket := strings.TrimSpace(ms.R2.Bucket)
	endpoint := strings.TrimSpace(ms.R2.Endpoint)
	if accessKey == "" || secretKey == "" || bucket == "" || endpoint == "" {
		return
	}
	region := strings.TrimSpace(ms.R2.Region)
	if region == "" {
		region = "auto"
	}
	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	out.R2Client = client
	out.R2BucketName = bucket
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
	attachR2FromSetting(out, ms)
	if err := attachBunnyStorageFromSetting(out, ms); err != nil {
		return nil, err
	}
	return out, nil
}
