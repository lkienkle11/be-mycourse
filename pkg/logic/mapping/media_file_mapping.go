package mapping

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/entities"
)

func ToUploadFileResponse(file entities.File) dto.UploadFileResponse {
	return dto.UploadFileResponse{
		URL:       file.URL,
		OriginURL: file.OriginURL,
		ObjectKey: file.ObjectKey,
		Metadata:  file.Metadata,
	}
}

func ToUploadFileResponses(files []entities.File) []dto.UploadFileResponse {
	out := make([]dto.UploadFileResponse, 0, len(files))
	for _, item := range files {
		out = append(out, ToUploadFileResponse(item))
	}
	return out
}
