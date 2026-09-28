package application

import (
	"context"
	"fmt"
	"time"

	"mycourse-io-be/internal/instructor/domain"
	"mycourse-io-be/internal/shared/cache"
)

const popularInstructorsCacheTTL = 5 * time.Minute

func popularInstructorsCacheKey(limit int) string {
	return fmt.Sprintf("mycourse:catalog:popular_instructors:limit:%d", limit)
}

// ListPopularInstructors is cache-aside (5 min TTL) — see
// openspec/changes/add-home-catalog-apis/design.md Decision #3. Fails open:
// a cache miss or unavailable Redis always falls through to the repository.
func (s *InstructorService) ListPopularInstructors(ctx context.Context, limit int) ([]domain.PopularInstructor, error) {
	key := popularInstructorsCacheKey(limit)
	if cached, ok := cache.GetJSON[[]domain.PopularInstructor](ctx, key); ok {
		return cached, nil
	}
	items, err := s.repo.ListPopularInstructors(ctx, limit)
	if err != nil {
		return nil, err
	}
	cache.SetJSON(ctx, key, popularInstructorsCacheTTL, items)
	return items, nil
}
