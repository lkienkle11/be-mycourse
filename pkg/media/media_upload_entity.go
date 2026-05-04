package media

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/pkg/logic/utils"
)

func mergeUploadInputMetadata(in entities.MediaUploadEntityInput) entities.RawMetadata {
	uploadedMeta := NormalizeMetadata(in.Uploaded.Metadata)
	merged := NormalizeMetadata(in.UploadedMeta)
	for k, v := range uploadedMeta {
		merged[k] = v
	}
	return merged
}

func streamMetadataFromMerged(merged entities.RawMetadata, typed entities.UploadFileMetadata) (
	bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider string,
	duration int64,
) {
	bunnyVideoID = strings.TrimSpace(fmt.Sprintf("%v", merged["bunny_video_id"]))
	if bunnyVideoID == "" {
		bunnyVideoID = strings.TrimSpace(fmt.Sprintf("%v", merged["video_guid"]))
	}
	videoID = strings.TrimSpace(fmt.Sprintf("%v", merged[constants.MediaMetaKeyVideoID]))
	if videoID == "" {
		videoID = bunnyVideoID
	}
	thumbnailURL = strings.TrimSpace(fmt.Sprintf("%v", merged[constants.MediaMetaKeyThumbnailURL]))
	embededHTML = strings.TrimSpace(fmt.Sprintf("%v", merged[constants.MediaMetaKeyEmbededHTML]))
	bunnyLibraryID = strings.TrimSpace(fmt.Sprintf("%v", merged["bunny_library_id"]))
	videoProvider = strings.TrimSpace(fmt.Sprintf("%v", merged["video_provider"]))
	duration = int64(typed.DurationSeconds)
	if duration <= 0 {
		duration = int64(utils.FloatFromRaw(merged, "length"))
	}
	return bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider, duration
}

func b2BucketFromUploadInput(in entities.MediaUploadEntityInput, merged entities.RawMetadata) string {
	b2Bucket := strings.TrimSpace(in.B2Bucket)
	if b2Bucket == "" {
		b2Bucket = strings.TrimSpace(fmt.Sprintf("%v", merged["b2_bucket_name"]))
	}
	return b2Bucket
}

func preservedOrNewEntityID(in entities.MediaUploadEntityInput) string {
	id := strings.TrimSpace(in.PreserveID)
	if in.GenerateNewID || id == "" {
		return uuid.NewString()
	}
	return id
}

func newFileEntityUploadCore(in entities.MediaUploadEntityInput, merged entities.RawMetadata, typed entities.UploadFileMetadata) *entities.File {
	return &entities.File{
		ID:                 preservedOrNewEntityID(in),
		Kind:               in.Kind,
		Provider:           in.Provider,
		Filename:           in.Filename,
		MimeType:           in.ContentType,
		SizeBytes:          in.SizeBytes,
		URL:                in.Uploaded.URL,
		OriginURL:          in.Uploaded.OriginURL,
		ObjectKey:          in.Uploaded.ObjectKey,
		Status:             constants.FileStatusReady,
		B2BucketName:       b2BucketFromUploadInput(in, merged),
		Metadata:           typed,
		CreatedAt:          in.CreatedAt,
		UpdatedAt:          in.UpdatedAt,
		RowVersion:         1,
		ContentFingerprint: "",
	}
}

func attachStreamFieldsToFile(
	f *entities.File,
	bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider string,
	duration int64,
) {
	f.BunnyVideoID = bunnyVideoID
	f.BunnyLibraryID = bunnyLibraryID
	f.VideoID = videoID
	f.ThumbnailURL = thumbnailURL
	f.EmbededHTML = embededHTML
	f.Duration = duration
	f.VideoProvider = videoProvider
}

func fileEntityFromUploadStreamFields(
	in entities.MediaUploadEntityInput,
	merged entities.RawMetadata,
	typed entities.UploadFileMetadata,
	bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider string,
	duration int64,
) *entities.File {
	f := newFileEntityUploadCore(in, merged, typed)
	attachStreamFieldsToFile(f, bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider, duration)
	return f
}

func BuildMediaFileEntityFromUpload(in entities.MediaUploadEntityInput) *entities.File {
	merged := mergeUploadInputMetadata(in)
	typed := BuildTypedMetadata(in.Kind, in.ContentType, in.Filename, in.SizeBytes, in.Payload, merged)
	bv, vid, thumb, embed, lib, vprov, dur := streamMetadataFromMerged(merged, typed)
	return fileEntityFromUploadStreamFields(in, merged, typed, bv, vid, thumb, embed, lib, vprov, dur)
}
