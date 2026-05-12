package infra

import (
	"encoding/json"
	"strings"
)

// ScanJSONBForImageURLs walks a raw JSONB payload (unmarshalled as map[string]any
// or []any) and collects all string values that look like image/media URLs.
//
// This helper is intentionally generic. It is designed for use by future domain
// services (section/lesson content JSONB, quiz option images, course banner arrays)
// when they need to enumerate orphan URLs before a cascade delete.
//
// Keys treated as URL fields (case-insensitive suffix match):
//
//	"_url", "image", "thumbnail", "cover", "banner", "avatar", "poster", "icon"
//
// TODO (Phase 05/06 domain — sections/lessons/quiz):
//
//	When lesson content JSONB, quiz option images, or course_edit metadata are
//	added, call ScanJSONBForImageURLs on the stored JSONB payload, then pass each
//	collected URL to internal/jobs/media.EnqueueOrphanImageCleanup AFTER the parent DB row
//	has been successfully deleted.
//
//	Example integration point (pseudo-code):
//
//	  func DeleteLesson(id uint) error {
//	      row, _ := lessonRepo.GetByID(id)
//	      if err := lessonRepo.Delete(id); err != nil { return err }
//	      for _, u := range helper.ScanJSONBForImageURLs(row.ContentJSON) {
//	          internal/jobs/media.EnqueueOrphanImageCleanup(u)
//	      }
//	      return nil
//	  }
func ScanJSONBForImageURLs(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	out := make([]string, 0, 4)
	scanValue(v, &out)
	return out
}

func scanValue(v any, out *[]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			lk := strings.ToLower(k)
			if isURLKey(lk) {
				if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
					*out = append(*out, strings.TrimSpace(s))
				}
			} else {
				scanValue(val, out)
			}
		}
	case []any:
		for _, item := range t {
			scanValue(item, out)
		}
	}
}

// imageURLKeySuffixes are key name patterns that imply the value is an image/media URL.
var imageURLKeySuffixes = []string{
	"_url", "image", "thumbnail", "cover", "banner", "avatar", "poster", "icon",
}

func isURLKey(key string) bool {
	for _, suffix := range imageURLKeySuffixes {
		if strings.HasSuffix(key, suffix) || key == suffix {
			return true
		}
	}
	return false
}
