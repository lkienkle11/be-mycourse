package infra

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/utils"
)

// serializeMergedMetadataJSON marshals the merged raw metadata map into a
// JSON string that will be stored in the media_files.metadata_json JSONB
// column. Returns "{}" when there is nothing to persist or when marshalling
// fails (defensive: never break persistence on a serialisation error).
func serializeMergedMetadataJSON(merged domain.RawMetadata) string {
	if len(merged) == 0 {
		return "{}"
	}
	blob, err := json.Marshal(merged)
	if err != nil {
		return "{}"
	}
	return string(blob)
}

func mergeUploadInputMetadata(in domain.MediaUploadEntityInput) domain.RawMetadata {
	uploadedMeta := NormalizeMetadata(in.Uploaded.Metadata)
	merged := NormalizeMetadata(in.UploadedMeta)
	for k, v := range uploadedMeta {
		merged[k] = v
	}
	return merged
}

func streamMetadataFromMerged(merged domain.RawMetadata, typed domain.UploadFileMetadata) (
	bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider string,
	duration int64,
) {
	bunnyVideoID = strings.TrimSpace(fmt.Sprintf("%v", merged["bunny_video_id"]))
	if bunnyVideoID == "" {
		bunnyVideoID = strings.TrimSpace(fmt.Sprintf("%v", merged["video_guid"]))
	}
	videoID = strings.TrimSpace(fmt.Sprintf("%v", merged[domain.MediaMetaKeyVideoID]))
	if videoID == "" {
		videoID = bunnyVideoID
	}
	thumbnailURL = strings.TrimSpace(fmt.Sprintf("%v", merged[domain.MediaMetaKeyThumbnailURL]))
	embededHTML = strings.TrimSpace(fmt.Sprintf("%v", merged[domain.MediaMetaKeyEmbededHTML]))
	bunnyLibraryID = strings.TrimSpace(fmt.Sprintf("%v", merged["bunny_library_id"]))
	videoProvider = strings.TrimSpace(fmt.Sprintf("%v", merged["video_provider"]))
	duration = int64(typed.DurationSeconds)
	if duration <= 0 {
		duration = int64(utils.FloatFromRaw(merged, "length"))
	}
	return bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider, duration
}

func b2BucketFromUploadInput(in domain.MediaUploadEntityInput, merged domain.RawMetadata) string {
	b2Bucket := strings.TrimSpace(in.B2Bucket)
	if b2Bucket == "" {
		b2Bucket = strings.TrimSpace(fmt.Sprintf("%v", merged["b2_bucket_name"]))
	}
	return b2Bucket
}

func preservedOrNewEntityID(in domain.MediaUploadEntityInput) string {
	id := strings.TrimSpace(in.PreserveID)
	if in.GenerateNewID || id == "" {
		return uuid.NewString()
	}
	return id
}

func newFileEntityUploadCore(in domain.MediaUploadEntityInput, merged domain.RawMetadata, typed domain.UploadFileMetadata) *domain.File {
	return &domain.File{
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
		// Persist the merged provider metadata into the JSONB column.
		// Without this assignment the database row would always store "{}"
		// (see fileToRow in repos.go) even though Bunny/B2 return useful
		// fields like length, framerate, resolution, thumbnail_filename, ...
		MetadataJSON:       serializeMergedMetadataJSON(merged),
		CreatedAt:          in.CreatedAt,
		UpdatedAt:          in.UpdatedAt,
		RowVersion:         1,
		ContentFingerprint: "",
	}
}

func attachStreamFieldsToFile(
	f *domain.File,
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
	in domain.MediaUploadEntityInput,
	merged domain.RawMetadata,
	typed domain.UploadFileMetadata,
	bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider string,
	duration int64,
) *domain.File {
	f := newFileEntityUploadCore(in, merged, typed)
	attachStreamFieldsToFile(f, bunnyVideoID, videoID, thumbnailURL, embededHTML, bunnyLibraryID, videoProvider, duration)
	return f
}

func BuildMediaFileEntityFromUpload(in domain.MediaUploadEntityInput) *domain.File {
	merged := mergeUploadInputMetadata(in)
	typed := BuildTypedMetadata(in.Kind, in.ContentType, in.Filename, in.SizeBytes, in.Payload, merged)
	bv, vid, thumb, embed, lib, vprov, dur := streamMetadataFromMerged(merged, typed)
	return fileEntityFromUploadStreamFields(in, merged, typed, bv, vid, thumb, embed, lib, vprov, dur)
}
