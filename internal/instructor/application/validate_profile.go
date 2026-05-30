package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/instructor/domain"
)

func (s *InstructorService) validateProfile(ctx context.Context, p domain.ProfilePayload) error {
	if s.mediaVal == nil {
		return nil
	}
	return s.mediaVal.ValidateProfilePayload(ctx, p)
}

func normalizeRejectionReason(reason string) (string, error) {
	r := strings.TrimSpace(reason)
	if len(r) < 1 || len(r) > 2000 {
		return "", domain.ErrRejectionReasonRequired
	}
	return r, nil
}
