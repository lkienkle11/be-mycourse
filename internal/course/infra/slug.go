package infra

import (
	"context"
	"fmt"
	"strings"

	gosimpleslug "github.com/gosimple/slug"
	"gorm.io/gorm"

	"mycourse-io-be/internal/course/domain"
	sharedslug "mycourse-io-be/internal/shared/slug"
)

// buildSuffixedCandidate truncates base to leave room for "-"+suffix within
// domain.MaxSlugLen and returns an error if no room remains at all (base
// truncates to empty). Code review found the untruncated/unguarded version
// used by an earlier draft could otherwise produce a leading-hyphen candidate
// (e.g. "-x7k92ab") once escalating suffix length consumed the entire column
// budget — a value that violates the slug format itself, not just a
// collision. Shared by all 3 RetryWithSuffix call sites in this file so the
// guard exists exactly once.
func buildSuffixedCandidate(base, suffix string) (string, error) {
	truncatedBase := sharedslug.TruncateForSuffix(base, domain.MaxSlugLen, len(suffix))
	if truncatedBase == "" {
		return "", fmt.Errorf("slug base %q leaves no room for a %d-character suffix within %d characters", base, len(suffix), domain.MaxSlugLen)
	}
	return truncatedBase + "-" + suffix, nil
}

// generateAutoSlugBase derives a base slug from title via gosimple/slug,
// truncated to domain.MaxSlugLen BEFORE format validation (a long or
// multi-byte title — e.g. Chinese/Japanese, where each character can
// transliterate to several ASCII characters plus a separating hyphen — can
// produce a transliteration far longer than the original title). An empty or
// still-format-invalid result after truncation falls back to the literal
// base "course" and signals the caller to skip trying it bare (see
// resolveCreateSlug in repo_instructor.go).
func generateAutoSlugBase(title string) (base string, mustSuffix bool) {
	candidate := gosimpleslug.Make(title)
	candidate = sharedslug.TruncateToLen(candidate, domain.MaxSlugLen)
	if trimmed, ok := sharedslug.ValidateManualFormat(candidate, domain.MaxSlugLen); ok {
		return trimmed, false
	}
	return "course", true
}

// courseSlugAvailable reports whether slug is free among active courses,
// excluding excludeCourseID (used on update to allow "no-op" resubmission of
// the current slug without treating it as a self-collision).
func courseSlugAvailable(ctx context.Context, db *gorm.DB, slug string, excludeCourseID *string) (bool, error) {
	q := db.WithContext(ctx).Model(&courseRow{}).Where("deleted_at IS NULL").Where("slug = ?", slug)
	if excludeCourseID != nil {
		if id := strings.TrimSpace(*excludeCourseID); id != "" {
			q = q.Where("id != ?", id)
		}
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}
