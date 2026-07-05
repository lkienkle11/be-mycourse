package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
)

func hydrateProfilePayloadMedia(
	ctx context.Context,
	hydr MediaHydrator,
	p *domain.ProfilePayload,
	setCVFile func(*domain.MediaFileReadModel),
	setIntroVideoFile func(*domain.MediaFileReadModel),
) error {
	if hydr == nil || p == nil {
		return nil
	}
	ids := make([]string, 0, 2+len(p.Certificates))
	if id := strings.TrimSpace(p.CVFileID); id != "" {
		ids = append(ids, id)
	}
	if id := strings.TrimSpace(p.IntroVideoFileID); id != "" {
		ids = append(ids, id)
	}
	for _, cert := range p.Certificates {
		if id := strings.TrimSpace(cert.CertificateFileID); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	files, err := hydr.ResolveMediaFiles(ctx, ids)
	if err != nil {
		return err
	}
	if f, ok := files[strings.TrimSpace(p.CVFileID)]; ok && setCVFile != nil {
		copy := f
		setCVFile(&copy)
	}
	if f, ok := files[strings.TrimSpace(p.IntroVideoFileID)]; ok && setIntroVideoFile != nil {
		copy := f
		setIntroVideoFile(&copy)
	}
	for i := range p.Certificates {
		id := strings.TrimSpace(p.Certificates[i].CertificateFileID)
		if f, ok := files[id]; ok {
			copy := f
			p.Certificates[i].CertificateFile = &copy
		}
	}
	return nil
}
