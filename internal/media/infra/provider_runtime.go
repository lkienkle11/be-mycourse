package infra

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/setting"
)

// CloudClients stores initialized media provider SDK/HTTP clients.
type CloudClients struct {
	R2Client     *s3.Client
	R2BucketName string
	HTTPClient   *http.Client
}

var Cloud *CloudClients

// Setup initializes media SDK clients once at app startup.
func Setup() error {
	client, err := NewCloudClientsFromSetting()
	if err != nil {
		return fmt.Errorf(constants.MsgSetupMediaClientsFailed, err)
	}
	Cloud = client
	return nil
}

func RequireInitialized[T any](dependency *T) error {
	if dependency == nil {
		return apperrors.ErrDependencyNotConfigured
	}
	return nil
}

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

// NewCloudClientsFromSetting builds R2 (S3 API) handles from setting.MediaSetting.
func NewCloudClientsFromSetting() (*CloudClients, error) {
	out := &CloudClients{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
	attachR2FromSetting(out, setting.MediaSetting)
	return out, nil
}
