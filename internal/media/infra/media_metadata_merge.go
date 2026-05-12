package infra

import (
	"encoding/json"

	"mycourse-io-be/internal/media/domain"
)

func MergeMediaMetadataJSON(prevJSON []byte, overlay domain.RawMetadata) ([]byte, error) {
	base := map[string]any{}
	if len(prevJSON) > 0 {
		_ = json.Unmarshal(prevJSON, &base)
	}
	if base == nil {
		base = map[string]any{}
	}
	for k, v := range overlay {
		base[k] = v
	}
	return json.Marshal(base)
}
