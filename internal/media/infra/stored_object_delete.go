package infra

import (
	"context"
	"strings"

	"mycourse-io-be/internal/shared/constants"
)

func DeleteStoredObject(ctx context.Context, clients *CloudClients, objectKey string, provider string, bunnyVideoID string) error {
	switch provider {
	case constants.FileProviderLocal:
		return nil
	case constants.FileProviderBunny:
		guid := strings.TrimSpace(bunnyVideoID)
		if guid == "" {
			guid = strings.TrimSpace(objectKey)
		}
		return DeleteBunnyVideo(clients, ctx, guid)
	default:
		return DeleteB2Object(clients, ctx, objectKey)
	}
}
