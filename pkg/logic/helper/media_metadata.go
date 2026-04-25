package helper

import (
	"encoding/json"
	"strings"

	"mycourse-io-be/pkg/entities"
)

func ParseMetadataJSON(raw string) (entities.FileMetadata, error) {
	if strings.TrimSpace(raw) == "" {
		return entities.FileMetadata{}, nil
	}
	out := make(entities.FileMetadata)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return NormalizeMetadata(out), nil
}

func NormalizeMetadata(in map[string]any) entities.FileMetadata {
	if in == nil {
		return entities.FileMetadata{}
	}
	out := make(entities.FileMetadata, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = v
	}
	return out
}
