package media

import (
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/utils"
)

func readMultipartPayloadLimited(file multipart.File, fileHeader *multipart.FileHeader) (payload []byte, filename, mime string, err error) {
	filename = strings.TrimSpace(fileHeader.Filename)
	mime = fileHeader.Header.Get("Content-Type")
	if fileHeader.Size >= 0 && fileHeader.Size > constants.MaxMediaUploadFileBytes {
		return nil, filename, mime, pkgerrors.ErrFileExceedsMaxUploadSize
	}
	limited := io.LimitReader(file, constants.MaxMediaUploadFileBytes+1)
	payload, err = io.ReadAll(limited)
	if err != nil {
		return nil, filename, mime, err
	}
	if int64(len(payload)) > constants.MaxMediaUploadFileBytes {
		return nil, filename, mime, pkgerrors.ErrFileExceedsMaxUploadSize
	}
	return payload, filename, mime, nil
}

func rejectExecutableNonMedia(kind constants.FileKind, isImage bool, filename string, payload []byte) error {
	if kind != constants.FileKindFile || isImage {
		return nil
	}
	head := payload
	if len(head) > 16 {
		head = head[:16]
	}
	if utils.IsExecutableUploadRejected(filename, head) {
		return pkgerrors.ErrExecutableUploadRejected
	}
	return nil
}

func effectiveUploadSizeBytes(headerSize int64, payload []byte, isImage bool) int64 {
	if isImage {
		return int64(len(payload))
	}
	if headerSize < 0 || headerSize == 0 {
		return int64(len(payload))
	}
	return headerSize
}

func encodeUploadToWebP(payload []byte, filename string) ([]byte, string, string, error) {
	utils.AcquireEncodeGate()
	encoded, newMime, encErr := utils.EncodeWebP(payload)
	utils.ReleaseEncodeGate()
	if encErr != nil {
		return nil, "", "", &pkgerrors.ProviderError{
			Code: errcode.ImageEncodeBusy,
			Msg:  encErr.Error(),
			Err:  encErr,
		}
	}
	outName := filename
	if ext := filepath.Ext(filename); ext != "" {
		outName = strings.TrimSuffix(filename, ext) + ".webp"
	}
	return encoded, newMime, outName, nil
}
