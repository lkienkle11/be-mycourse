package media

import (
	"encoding/json"

	"mycourse-io-be/pkg/entities"
)

func MergeMediaMetadataJSON(prevJSON []byte, overlay entities.RawMetadata) ([]byte, error) {
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
