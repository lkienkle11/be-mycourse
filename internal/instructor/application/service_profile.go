package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
)

func (s *InstructorService) ListProfiles(ctx context.Context, f domain.ProfileFilter) ([]domain.Profile, int64, error) {
	f = normalizeProfileFilter(f)
	return listWithIdentity(
		s,
		ctx,
		func() ([]domain.Profile, int64, error) { return s.repo.ListProfiles(ctx, f) },
		profileAvatarFileID,
		setProfileAvatarURL,
	)
}

func (s *InstructorService) GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	row, err := loadOneWithIdentity(
		s,
		ctx,
		func() (*domain.Profile, error) { return s.repo.GetProfileByUserID(ctx, userID) },
		profileAvatarFileID,
		setProfileAvatarURL,
	)
	if err != nil {
		return nil, err
	}
	if err := s.hydrateProfileMedia(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *InstructorService) hydrateProfileMedia(ctx context.Context, profile *domain.Profile) error {
	if s.mediaHydr == nil || profile == nil {
		return nil
	}
	p := &profile.ProfilePayload
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
	files, err := s.mediaHydr.ResolveMediaFiles(ctx, ids)
	if err != nil {
		return err
	}
	if f, ok := files[strings.TrimSpace(p.CVFileID)]; ok {
		copy := f
		profile.CVFile = &copy
	}
	if f, ok := files[strings.TrimSpace(p.IntroVideoFileID)]; ok {
		copy := f
		profile.IntroVideoFile = &copy
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

func normalizeProfileFilter(f domain.ProfileFilter) domain.ProfileFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	return f
}

func (s *InstructorService) UpsertProfile(ctx context.Context, in domain.UpsertProfileInput) (*domain.Profile, error) {
	if err := s.validateProfile(ctx, in.ProfilePayload); err != nil {
		return nil, err
	}
	return s.repo.UpsertProfile(ctx, in)
}

func (s *InstructorService) DeleteProfile(ctx context.Context, userID string) error {
	return s.repo.DeleteProfileByUserID(ctx, userID)
}
