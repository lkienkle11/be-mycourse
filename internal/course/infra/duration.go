package infra

import (
	"encoding/json"
	"math"
	"strings"

	"mycourse-io-be/internal/course/domain"
	sharedutils "mycourse-io-be/internal/shared/utils"
)

const maxEstimatedDurationMs = int64(999) * 3600 * 1000

func resolveSubLessonEstimatedDurationMs(sub domain.SubLesson, mediaMs int64) int64 {
	switch strings.ToUpper(strings.TrimSpace(sub.Kind)) {
	case domain.SubLessonKindVideo:
		if mediaMs < 0 {
			return 0
		}
		return mediaMs
	default:
		if sub.EstimatedDurationMs < 0 {
			return 0
		}
		return sub.EstimatedDurationMs
	}
}

func normalizeSubLessonEstimatedDurationMs(kind string, inputMs int64) (int64, error) {
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if kind == domain.SubLessonKindVideo {
		return 0, nil
	}
	if inputMs < 0 || inputMs > maxEstimatedDurationMs {
		return 0, domain.ErrCourseInvalidSubLessonKind
	}
	return inputMs, nil
}

func collectVideoMediaFileIDs(sections []domain.Section) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, section := range sections {
		for _, lesson := range section.Lessons {
			for _, sub := range lesson.SubLessons {
				if sub.Kind != domain.SubLessonKindVideo || sub.Video == nil {
					continue
				}
				fileID := strings.TrimSpace(sub.Video.MediaFileID)
				if fileID == "" {
					continue
				}
				if _, ok := seen[fileID]; ok {
					continue
				}
				seen[fileID] = struct{}{}
				ids = append(ids, fileID)
			}
		}
	}
	return ids
}

func applyOutlineEstimatedDurations(sections []domain.Section, videoMediaMs map[string]int64) {
	for i := range sections {
		var sectionTotal int64
		for j := range sections[i].Lessons {
			var lessonTotal int64
			for k := range sections[i].Lessons[j].SubLessons {
				sub := &sections[i].Lessons[j].SubLessons[k]
				var mediaMs int64
				if sub.Kind == domain.SubLessonKindVideo && sub.Video != nil {
					mediaMs = videoMediaMs[strings.TrimSpace(sub.Video.MediaFileID)]
				}
				resolved := resolveSubLessonEstimatedDurationMs(*sub, mediaMs)
				sub.EstimatedDurationMs = resolved
				lessonTotal += resolved
			}
			sections[i].Lessons[j].EstimatedDurationMs = lessonTotal
			sectionTotal += lessonTotal
		}
		sections[i].EstimatedDurationMs = sectionTotal
	}
}

func applySubLessonListEstimatedDurations(subs []domain.SubLesson, videoMediaMs map[string]int64) {
	for i := range subs {
		var mediaMs int64
		if subs[i].Kind == domain.SubLessonKindVideo && subs[i].Video != nil {
			mediaMs = videoMediaMs[strings.TrimSpace(subs[i].Video.MediaFileID)]
		}
		subs[i].EstimatedDurationMs = resolveSubLessonEstimatedDurationMs(subs[i], mediaMs)
	}
}

// mediaDurationSecondsFromStored resolves video length in seconds from the
// media_files row: prefer the duration column, then metadata_json keys used by
// the media module (duration_seconds, duration, length).
func mediaDurationSecondsFromStored(durationCol int64, metadataJSON []byte) int64 {
	if durationCol > 0 {
		return durationCol
	}
	if len(metadataJSON) == 0 {
		return 0
	}
	var raw map[string]any
	if err := json.Unmarshal(metadataJSON, &raw); err != nil || len(raw) == 0 {
		return 0
	}
	sec := sharedutils.FloatFromRaw(raw, "duration_seconds")
	if sec <= 0 {
		sec = sharedutils.FloatFromRaw(raw, "duration")
	}
	if sec <= 0 {
		sec = sharedutils.FloatFromRaw(raw, "length")
	}
	if sec <= 0 {
		return 0
	}
	return int64(math.Floor(sec + 1e-9))
}

func durationSecondsToMs(seconds int64) int64 {
	if seconds <= 0 {
		return 0
	}
	return seconds * 1000
}
