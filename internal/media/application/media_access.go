package application

import (
	"strings"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
)

func normalizeMediaVisibility(raw string) string {
	if strings.TrimSpace(raw) == constants.MediaVisibilityPublic {
		return constants.MediaVisibilityPublic
	}
	return constants.MediaVisibilityPrivate
}

// NormalizeMediaVisibility is the exported alias for multipart visibility parsing.
func NormalizeMediaVisibility(raw string) string {
	return normalizeMediaVisibility(raw)
}

func canViewMediaFile(file *domain.File, viewerUserID string) bool {
	if file == nil {
		return false
	}
	if file.Visibility == constants.MediaVisibilityPublic {
		return true
	}
	ownerID := strings.TrimSpace(file.UserID)
	if ownerID == "" {
		return true
	}
	viewerUserID = strings.TrimSpace(viewerUserID)
	if viewerUserID == "" {
		return false
	}
	return ownerID == viewerUserID
}

func canMutateMediaFile(file *domain.File, viewerUserID string) bool {
	if file == nil {
		return false
	}
	ownerID := strings.TrimSpace(file.UserID)
	if ownerID == "" {
		return false
	}
	viewerUserID = strings.TrimSpace(viewerUserID)
	if viewerUserID == "" {
		return false
	}
	return ownerID == viewerUserID
}
