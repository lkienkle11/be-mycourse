// Package mediaquery resolves media file IDs to URLs without importing media/domain.
package mediaquery

import "context"

// FileURLResolver loads public URLs for media file IDs.
type FileURLResolver interface {
	URLsForFileIDs(ctx context.Context, fileIDs []string) (map[string]string, error)
}

// HydrateAvatarURLs returns fileID → URL for roster avatars.
func HydrateAvatarURLs(ctx context.Context, resolver FileURLResolver, fileIDs []string) (map[string]string, error) {
	if resolver == nil || len(fileIDs) == 0 {
		return map[string]string{}, nil
	}
	return resolver.URLsForFileIDs(ctx, fileIDs)
}
