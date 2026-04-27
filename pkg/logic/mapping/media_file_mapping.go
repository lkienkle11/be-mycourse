package mapping

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/entities"
)

func ToUploadFileResponse(file entities.File) dto.UploadFileResponse {
	return dto.UploadFileResponse{
		ID:             file.ID,
		Kind:           string(file.Kind),
		Filename:       file.Filename,
		MimeType:       file.MimeType,
		SizeBytes:      file.SizeBytes,
		Status:         string(file.Status),
		B2BucketName:   file.B2BucketName,
		URL:            file.URL,
		OriginURL:      file.OriginURL,
		ObjectKey:      file.ObjectKey,
		BunnyVideoID:   file.BunnyVideoID,
		BunnyLibraryID: file.BunnyLibraryID,
		Duration:       file.Duration,
		VideoProvider:  file.VideoProvider,
		Metadata:       file.Metadata,
	}
}

func ToUploadFileResponses(files []entities.File) []dto.UploadFileResponse {
	out := make([]dto.UploadFileResponse, 0, len(files))
	for _, item := range files {
		out = append(out, ToUploadFileResponse(item))
	}
	return out
}
