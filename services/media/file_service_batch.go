package media

import (
	"context"
	"encoding/json"
	"time"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/logic/mapping"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	mediarepo "mycourse-io-be/repository/media"
)

func persistPreparedCreates(clients *entities.CloudClients, prepared []entities.PreparedCreatePart, uploaded []entities.ProviderUploadResult) ([]*entities.File, error) {
	if len(prepared) == 0 {
		return nil, nil
	}
	out := make([]*entities.File, len(prepared))
	now := time.Now()
	for i := range prepared {
		input := createFileEntityInput(prepared[i].Header, prepared[i].Payload, prepared[i].Filename, prepared[i].Mime, prepared[i].Kind, prepared[i].Provider, uploaded[i], now)
		ent, err := persistCreateMediaRow(clients, input, prepared[i].Payload)
		if err != nil {
			rollbackCreatedMediaRows(clients, out[:i])
			return nil, err
		}
		out[i] = ent
	}
	return out, nil
}

func rollbackCreatedMediaRows(clients *entities.CloudClients, rows []*entities.File) {
	for _, ent := range rows {
		if ent == nil {
			continue
		}
		_ = pkgmedia.DeleteStoredObject(context.Background(), clients, ent.ObjectKey, ent.Provider, ent.BunnyVideoID)
		_ = mediaRepository().SoftDeleteByObjectKey(ent.ObjectKey)
	}
}

// CreateFiles uploads up to MaxMediaFilesPerRequest parts in one request (parallel provider upload,
// sequential DB persist). All-or-nothing: any failure rolls back prior persisted rows for this call.
func CreateFiles(req dto.CreateFileRequest, parts []entities.OpenedUploadPart) ([]*entities.File, error) {
	if len(parts) == 0 {
		return nil, pkgerrors.ErrMediaFilesRequired
	}
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return nil, err
	}
	clients := pkgmedia.Cloud
	remaining := constants.MaxMediaMultipartTotalBytes
	prepared, err := mapping.PrepareCreatePartsSequential(req, parts, &remaining, prepareCreateMultipartBody)
	if err != nil {
		return nil, err
	}
	uploaded, err := uploadPreparedCreatesParallel(clients, prepared)
	if err != nil {
		return nil, err
	}
	return persistPreparedCreates(clients, prepared, uploaded)
}

func prepareUpdateBundleHead(repo *mediarepo.FileRepository, prevRow *models.MediaFile, req dto.UpdateFileRequest, part entities.OpenedUploadPart, remaining *int64) (*entities.File, *entities.PreparedUpdateHead, error) {
	return mapping.PrepareUpdateBundleHead(repo, prevRow, req, part, remaining, mapping.UpdateBundleHeadDeps{
		ReadPayload: func(p entities.OpenedUploadPart, rem *int64) ([]byte, string, string, error) {
			return readMultipartPayloadLimited(p.File, p.Header, rem)
		},
		ContentFingerprint: utils.ContentFingerprint,
		SaveUnchangedFingerprint: func(r any, prev *models.MediaFile, filename string, rowVersion int64) (*entities.File, error) {
			return saveUnchangedFingerprintMetadata(r.(*mediarepo.FileRepository), prev, filename, rowVersion)
		},
		NormalizeUpdate: normalizeUpdateMultipartPayload,
	})
}

func persistUpdatedHeadFromPrepared(clients *entities.CloudClients, repo *mediarepo.FileRepository, prevRow *models.MediaFile, head *entities.PreparedUpdateHead, uploaded entities.ProviderUploadResult) (*entities.File, error) {
	prevRaw := entities.RawMetadata{}
	_ = json.Unmarshal(prevRow.MetadataJSON, &prevRaw)
	merged := mergeProviderMetadataWithPrevious(uploaded, prevRaw)
	isImage := pkgmedia.IsImageMIMEOrExt(head.MimeNorm, head.FilenameNorm)
	sizeBytes := effectiveUploadSizeBytes(head.Header.Size, head.PayloadNorm, isImage)
	input := mediaUploadEntityInputForRowUpdate(prevRow, head.Kind, head.Provider, head.FilenameNorm, head.MimeNorm, sizeBytes, head.PayloadNorm, uploaded, merged)
	return persistUpdatedMediaRow(clients, repo, prevRow, input, head.PayloadNorm, head.Fingerprint)
}

func prepareOptionalTailPrepared(createReq dto.CreateFileRequest, parts []entities.OpenedUploadPart, remaining *int64) ([]entities.PreparedCreatePart, error) {
	return mapping.PrepareOptionalTailPrepared(createReq, parts, remaining, prepareCreateMultipartBody)
}

func loadUpdateBundleBase(objectKey string, req dto.UpdateFileRequest, parts []entities.OpenedUploadPart) (*mediarepo.FileRepository, *models.MediaFile, *entities.CloudClients, []*entities.File, error) {
	repoAny, prevRow, clients, out, err := mapping.LoadUpdateBundleBase(objectKey, req, parts, mapping.LoadUpdateBundleDeps{
		LoadTarget: func(ok string, rq dto.UpdateFileRequest) (any, *models.MediaFile, error) {
			return loadUpdateFileTarget(ok, rq)
		},
		RequireMedia: func() error {
			return pkgmedia.RequireInitialized(pkgmedia.Cloud)
		},
		GetClients: func() *entities.CloudClients {
			return pkgmedia.Cloud
		},
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return repoAny.(*mediarepo.FileRepository), prevRow, clients, out, nil
}

// UpdateFileBundle updates the row at objectKey with parts[0] and creates additional rows for parts[1:]
// (bundle upload). Persist order: tail creates first, then primary row update — rolls back tail on head failure.
func UpdateFileBundle(objectKey string, req dto.UpdateFileRequest, createReq dto.CreateFileRequest, parts []entities.OpenedUploadPart) ([]*entities.File, error) {
	repo, prevRow, clients, out, err := loadUpdateBundleBase(objectKey, req, parts)
	if err != nil {
		return nil, err
	}

	remaining := constants.MaxMediaMultipartTotalBytes

	skipHead, headPrep, err := prepareUpdateBundleHead(repo, prevRow, req, parts[0], &remaining)
	if err != nil {
		return nil, err
	}

	var tailPrepared []entities.PreparedCreatePart
	tailPrepared, err = prepareOptionalTailPrepared(createReq, parts, &remaining)
	if err != nil {
		return nil, err
	}

	if skipHead != nil {
		return finishUpdateBundleSkipHead(out, clients, tailPrepared, skipHead)
	}

	headUploaded, tailUploaded, err := uploadBundleParallel(clients, headPrep, tailPrepared)
	if err != nil {
		return nil, err
	}
	if err := persistBundleAfterParallelUpload(clients, repo, prevRow, headPrep, tailPrepared, headUploaded, tailUploaded, out); err != nil {
		return nil, err
	}
	return out, nil
}

func persistBundleAfterParallelUpload(
	clients *entities.CloudClients,
	repo *mediarepo.FileRepository,
	prevRow *models.MediaFile,
	headPrep *entities.PreparedUpdateHead,
	tailPrepared []entities.PreparedCreatePart,
	headUploaded entities.ProviderUploadResult,
	tailUploaded []entities.ProviderUploadResult,
	out []*entities.File,
) error {
	tailEntities, err := persistPreparedCreates(clients, tailPrepared, tailUploaded)
	if err != nil {
		deleteUploadAttempt(clients, headPrep.Provider, headPrep.ResolvedObjectKey, headUploaded)
		return err
	}
	headEntity, err := persistUpdatedHeadFromPrepared(clients, repo, prevRow, headPrep, headUploaded)
	if err != nil {
		rollbackCreatedMediaRows(clients, tailEntities)
		deleteUploadAttempt(clients, headPrep.Provider, headPrep.ResolvedObjectKey, headUploaded)
		return err
	}
	fillBundleOut(out, headEntity, tailEntities)
	return nil
}

func fillBundleOut(out []*entities.File, headEntity *entities.File, tailEntities []*entities.File) {
	out[0] = headEntity
	for i := range tailEntities {
		out[1+i] = tailEntities[i]
	}
}

func finishUpdateBundleSkipHead(out []*entities.File, clients *entities.CloudClients, tailPrepared []entities.PreparedCreatePart, skipHead *entities.File) ([]*entities.File, error) {
	out[0] = skipHead
	if len(tailPrepared) == 0 {
		return out[:1], nil
	}
	uploadedTail, uerr := uploadPreparedCreatesParallel(clients, tailPrepared)
	if uerr != nil {
		return nil, uerr
	}
	tailEntities, perr := persistPreparedCreates(clients, tailPrepared, uploadedTail)
	if perr != nil {
		return nil, perr
	}
	for i := range tailEntities {
		out[1+i] = tailEntities[i]
	}
	return out, nil
}
