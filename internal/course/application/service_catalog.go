package application

import (
	"context"
	"fmt"
	"time"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/cache"
)

const trendingCoursesCacheTTL = 5 * time.Minute

func trendingCoursesCacheKey(limit int) string {
	return fmt.Sprintf("mycourse:catalog:trending_courses:limit:%d", limit)
}

// ListTrendingCourses is cache-aside (5 min TTL) — see
// openspec/changes/add-home-catalog-apis/design.md Decision #3. Fails open:
// a cache miss or unavailable Redis always falls through to the repository.
func (s *CourseService) ListTrendingCourses(ctx context.Context, limit int) ([]domain.TrendingCourseItem, error) {
	key := trendingCoursesCacheKey(limit)
	if cached, ok := cache.GetJSON[[]domain.TrendingCourseItem](ctx, key); ok {
		return cached, nil
	}
	items, err := s.repo.ListTrendingCourses(ctx, limit)
	if err != nil {
		return nil, err
	}
	cache.SetJSON(ctx, key, trendingCoursesCacheTTL, items)
	return items, nil
}

// ListContinueLearning is per-user data that mutates on every lesson
// interaction, so it is intentionally not cached (see design.md's Goals).
func (s *CourseService) ListContinueLearning(ctx context.Context, userID string, limit int) ([]domain.ContinueLearningItem, error) {
	return s.repo.ListContinueLearning(ctx, userID, limit)
}
