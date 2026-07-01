package infra

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

type r2ObjectPutter interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func effectiveR2Bucket(c *CloudClients) string {
	if b := strings.TrimSpace(setting.MediaSetting.R2.Bucket); b != "" {
		return b
	}
	return strings.TrimSpace(c.R2BucketName)
}

func UploadR2(c *CloudClients, ctx context.Context, objectKey, filename string, file io.Reader, meta domain.RawMetadata) (domain.ProviderUploadResult, error) {
	if c == nil || c.R2Client == nil {
		return domain.ProviderUploadResult{}, fmt.Errorf(constants.MsgR2ClientNotConfigured)
	}
	return uploadR2WithPutter(c, c.R2Client, ctx, objectKey, filename, file, meta)
}

func uploadR2WithPutter(
	c *CloudClients,
	putter r2ObjectPutter,
	ctx context.Context,
	objectKey, filename string,
	file io.Reader,
	meta domain.RawMetadata,
) (domain.ProviderUploadResult, error) {
	bucketName := effectiveR2Bucket(c)
	if bucketName == "" {
		return domain.ProviderUploadResult{}, &domain.ProviderError{Code: apperrors.R2BucketNotConfigured}
	}
	key := strings.TrimLeft(objectKey, "/")
	contentType := r2ObjectContentType(meta)
	if err := putR2Object(ctx, putter, bucketName, key, contentType, file); err != nil {
		return domain.ProviderUploadResult{}, err
	}
	return r2UploadResultURLs(bucketName, key, meta), nil
}

func putR2Object(ctx context.Context, putter r2ObjectPutter, bucketName, key, contentType string, body io.Reader) error {
	_, err := putter.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	return err
}

func r2ObjectContentType(meta domain.RawMetadata) string {
	if mt := strings.TrimSpace(utils.StringFromRaw(meta, domain.MediaMetaKeyMimeType)); mt != "" && !isBlockedStorageMIME(mt) {
		return mt
	}
	return "application/octet-stream"
}

func r2UploadResultURLs(bucketName, key string, meta domain.RawMetadata) domain.ProviderUploadResult {
	publicBase := utils.NormalizeBaseURL(setting.MediaSetting.R2.PublicURL, "")
	publicURL := utils.JoinURLPathSegments(publicBase, key)
	endpoint := utils.NormalizeBaseURL(setting.MediaSetting.R2.Endpoint, "")
	originURL := utils.JoinURLPathSegments(endpoint, bucketName, key)
	if meta == nil {
		meta = domain.RawMetadata{}
	}
	meta["r2_bucket_name"] = bucketName
	return domain.ProviderUploadResult{
		URL:       publicURL,
		OriginURL: originURL,
		ObjectKey: key,
		Metadata:  meta,
	}
}

func DeleteR2Object(c *CloudClients, ctx context.Context, objectKey string) error {
	if c.R2Client == nil {
		return fmt.Errorf(constants.MsgR2ClientNotConfigured)
	}
	bucketName := effectiveR2Bucket(c)
	if bucketName == "" {
		return &domain.ProviderError{Code: apperrors.R2BucketNotConfigured}
	}
	key := strings.TrimLeft(objectKey, "/")
	_, err := c.R2Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	return err
}

func buildR2PublicURL(objectKey string) string {
	publicBase := utils.NormalizeBaseURL(setting.MediaSetting.R2.PublicURL, "https://cdn.mycourse.local")
	key := strings.TrimLeft(objectKey, "/")
	return utils.JoinURLPathSegments(publicBase, key)
}
