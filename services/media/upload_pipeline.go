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

func readMultipartPayloadLimited(file multipart.File, fileHeader *multipart.FileHeader, remainingTotal *int64) (payload []byte, filename, mime string, err error) {
	filename = strings.TrimSpace(fileHeader.Filename)
	mime = fileHeader.Header.Get("Content-Type")

	perPartCap := multipartPerPartCap(remainingTotal)
	if perPartCap <= 0 {
		return nil, filename, mime, pkgerrors.ErrMediaMultipartTotalTooLarge
	}
	if err = validateMultipartDeclaredSizes(fileHeader, perPartCap); err != nil {
		return nil, filename, mime, err
	}

	limited := io.LimitReader(file, perPartCap+1)
	payload, err = io.ReadAll(limited)
	if err != nil {
		return nil, filename, mime, err
	}
	if err = validateReadPayloadSize(payload, perPartCap); err != nil {
		return nil, filename, mime, err
	}
	if remainingTotal != nil {
		*remainingTotal -= int64(len(payload))
	}
	return payload, filename, mime, nil
}

func multipartPerPartCap(remainingTotal *int64) int64 {
	perPartCap := constants.MaxMediaUploadFileBytes
	if remainingTotal != nil && *remainingTotal < perPartCap {
		perPartCap = *remainingTotal
	}
	return perPartCap
}

func validateMultipartDeclaredSizes(fileHeader *multipart.FileHeader, perPartCap int64) error {
	if fileHeader.Size >= 0 && fileHeader.Size > constants.MaxMediaUploadFileBytes {
		return pkgerrors.ErrFileExceedsMaxUploadSize
	}
	if fileHeader.Size >= 0 && fileHeader.Size > perPartCap {
		return pkgerrors.ErrMediaMultipartTotalTooLarge
	}
	return nil
}

func validateReadPayloadSize(payload []byte, perPartCap int64) error {
	if int64(len(payload)) > constants.MaxMediaUploadFileBytes {
		return pkgerrors.ErrFileExceedsMaxUploadSize
	}
	if int64(len(payload)) > perPartCap {
		return pkgerrors.ErrMediaMultipartTotalTooLarge
	}
	return nil
}

func rejectExecutableNonMedia(kind string, isImage bool, filename string, payload []byte) error {
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
