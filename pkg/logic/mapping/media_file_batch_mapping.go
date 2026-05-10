package mapping

import (
	"mime/multipart"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
)

// PrepareCreateMultipartBodyFn is the shape of services/media multipart prepare (injected to avoid mapping → services import).
type PrepareCreateMultipartBodyFn func(req dto.CreateFileRequest, file multipart.File, fileHeader *multipart.FileHeader, remainingTotal *int64) (
	payload []byte, filename, mime string, kind string, provider string, objectKey string, err error,
)

// PrepareCreatePartsSequential builds prepared create parts from opened multipart parts (Rule 14).
func PrepareCreatePartsSequential(req dto.CreateFileRequest, parts []entities.OpenedUploadPart, remaining *int64, prepareBody PrepareCreateMultipartBodyFn) ([]entities.PreparedCreatePart, error) {
	out := make([]entities.PreparedCreatePart, 0, len(parts))
	for _, p := range parts {
		payload, filename, mime, kind, provider, objectKey, err := prepareBody(req, p.File, p.Header, remaining)
		if err != nil {
			return nil, err
		}
		out = append(out, entities.PreparedCreatePart{
			Header:    p.Header,
			Payload:   payload,
			Filename:  filename,
			Mime:      mime,
			Kind:      kind,
			Provider:  provider,
			ObjectKey: objectKey,
		})
	}
	return out, nil
}

// PrepareOptionalTailPrepared prepares create parts for bundle tail segments parts[1:] (Rule 14).
func PrepareOptionalTailPrepared(createReq dto.CreateFileRequest, parts []entities.OpenedUploadPart, remaining *int64, prepareBody PrepareCreateMultipartBodyFn) ([]entities.PreparedCreatePart, error) {
	if len(parts) <= 1 {
		return nil, nil
	}
	return PrepareCreatePartsSequential(createReq, parts[1:], remaining, prepareBody)
}

// UpdateBundleHeadDeps injects I/O and fingerprint steps from services/media (Rule 14).
// Repo is typed as any so mapping does not import repository/ (depguard restrict_pkg).
type UpdateBundleHeadDeps struct {
	ReadPayload              func(part entities.OpenedUploadPart, remaining *int64) ([]byte, string, string, error)
	ContentFingerprint       func(payload []byte) string
	SaveUnchangedFingerprint func(repo any, prevRow *models.MediaFile, filename string, rowVersion int64) (*entities.File, error)
	NormalizeUpdate          func(filename, mime string, payload []byte) (
		payloadNorm []byte, filenameNorm, mimeNorm, kind, provider, resolvedObjectKey string, err error,
	)
}

func composePreparedUpdateHead(
	part entities.OpenedUploadPart,
	fp string,
	payload []byte,
	filename, mime string,
	payloadNorm []byte,
	filenameNorm, mimeNorm string,
	kind, provider, resolvedObjectKey string,
) *entities.PreparedUpdateHead {
	return &entities.PreparedUpdateHead{
		Header:            part.Header,
		Payload:           payload,
		Filename:          filename,
		Mime:              mime,
		Fingerprint:       fp,
		PayloadNorm:       payloadNorm,
		FilenameNorm:      filenameNorm,
		MimeNorm:          mimeNorm,
		Kind:              kind,
		Provider:          provider,
		ResolvedObjectKey: resolvedObjectKey,
	}
}

// PrepareUpdateBundleHead prepares the first bundle part (head) for update upload (Rule 14).
func PrepareUpdateBundleHead(
	repo any,
	prevRow *models.MediaFile,
	req dto.UpdateFileRequest,
	part entities.OpenedUploadPart,
	remaining *int64,
	deps UpdateBundleHeadDeps,
) (*entities.File, *entities.PreparedUpdateHead, error) {
	payload, filename, mime, err := deps.ReadPayload(part, remaining)
	if err != nil {
		return nil, nil, err
	}
	fp := deps.ContentFingerprint(payload)
	if req.SkipUploadIfUnchanged && prevRow.ContentFingerprint != "" && fp == prevRow.ContentFingerprint {
		ent, serr := deps.SaveUnchangedFingerprint(repo, prevRow, filename, prevRow.RowVersion)
		if serr != nil {
			return nil, nil, serr
		}
		return ent, nil, nil
	}
	payloadNorm, filenameNorm, mimeNorm, kind, provider, resolvedObjectKey, nerr := deps.NormalizeUpdate(filename, mime, payload)
	if nerr != nil {
		return nil, nil, nerr
	}
	head := composePreparedUpdateHead(part, fp, payload, filename, mime, payloadNorm, filenameNorm, mimeNorm, kind, provider, resolvedObjectKey)
	return nil, head, nil
}

// LoadUpdateBundleDeps loads repo row and cloud clients for bundle update (Rule 14).
type LoadUpdateBundleDeps struct {
	LoadTarget   func(objectKey string, req dto.UpdateFileRequest) (any, *models.MediaFile, error)
	RequireMedia func() error
	GetClients   func() *entities.CloudClients
}

// LoadUpdateBundleBase validates parts and loads repository state for UpdateFileBundle (Rule 14).
func LoadUpdateBundleBase(objectKey string, req dto.UpdateFileRequest, parts []entities.OpenedUploadPart, d LoadUpdateBundleDeps) (any, *models.MediaFile, *entities.CloudClients, []*entities.File, error) {
	if len(parts) == 0 {
		return nil, nil, nil, nil, pkgerrors.ErrMediaFilesRequired
	}
	repo, prevRow, err := d.LoadTarget(objectKey, req)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if err := d.RequireMedia(); err != nil {
		return nil, nil, nil, nil, err
	}
	out := make([]*entities.File, len(parts))
	return repo, prevRow, d.GetClients(), out, nil
}
