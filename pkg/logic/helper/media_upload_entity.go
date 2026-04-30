package helper

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/utils"
)

func BuildMediaFileEntityFromUpload(in entities.MediaUploadEntityInput) *entities.File {
	uploadedMeta := NormalizeMetadata(in.Uploaded.Metadata)
	merged := NormalizeMetadata(in.UploadedMeta)
	for k, v := range uploadedMeta {
		merged[k] = v
	}

	typedMetadata := BuildTypedMetadata(
		in.Kind,
		in.ContentType,
		in.Filename,
		in.SizeBytes,
		in.Payload,
		merged,
	)
	bunnyVideoID := strings.TrimSpace(fmt.Sprintf("%v", merged["bunny_video_id"]))
	if bunnyVideoID == "" {
		bunnyVideoID = strings.TrimSpace(fmt.Sprintf("%v", merged["video_guid"]))
	}
	bunnyLibraryID := strings.TrimSpace(fmt.Sprintf("%v", merged["bunny_library_id"]))
	videoProvider := strings.TrimSpace(fmt.Sprintf("%v", merged["video_provider"]))
	duration := int64(typedMetadata.DurationSeconds)
	if duration <= 0 {
		duration = int64(utils.FloatFromRaw(merged, "length"))
	}
	b2Bucket := strings.TrimSpace(in.B2Bucket)
	if b2Bucket == "" {
		b2Bucket = strings.TrimSpace(fmt.Sprintf("%v", merged["b2_bucket_name"]))
	}

	id := strings.TrimSpace(in.PreserveID)
	if in.GenerateNewID || id == "" {
		id = uuid.NewString()
	}

	return &entities.File{
		ID:                 id,
		Kind:               in.Kind,
		Provider:           in.Provider,
		Filename:           in.Filename,
		MimeType:           in.ContentType,
		SizeBytes:          in.SizeBytes,
		URL:                in.Uploaded.URL,
		OriginURL:          in.Uploaded.OriginURL,
		ObjectKey:          in.Uploaded.ObjectKey,
		Status:             constants.FileStatusReady,
		B2BucketName:       b2Bucket,
		BunnyVideoID:       bunnyVideoID,
		BunnyLibraryID:     bunnyLibraryID,
		Duration:           duration,
		VideoProvider:      videoProvider,
		Metadata:           typedMetadata,
		CreatedAt:          in.CreatedAt,
		UpdatedAt:          in.UpdatedAt,
		RowVersion:         1,
		ContentFingerprint: "",
	}
}
