package infra

import (
	"strings"
	"testing"
)

// TestTrendingCoursesQueryOrdersByCreatedAtNotID guards against repeating
// ListPublishedCourses's ORDER BY c.id DESC bug (id is a random UUID, not
// recency order) — trending must sort by created_at.
func TestTrendingCoursesQueryOrdersByCreatedAtNotID(t *testing.T) {
	if !strings.Contains(trendingCoursesQuery, "ORDER BY c.created_at DESC") {
		t.Fatalf("expected trendingCoursesQuery to ORDER BY c.created_at DESC, got:\n%s", trendingCoursesQuery)
	}
	if strings.Contains(trendingCoursesQuery, "ORDER BY c.id DESC") {
		t.Fatalf("trendingCoursesQuery must not repeat ListPublishedCourses's c.id DESC ordering bug")
	}
}

func TestTrendingCoursesQueryFiltersDraftAndTrashed(t *testing.T) {
	if !strings.Contains(trendingCoursesQuery, "c.deleted_at IS NULL") {
		t.Fatalf("expected trendingCoursesQuery to filter c.deleted_at IS NULL")
	}
	if !strings.Contains(trendingCoursesQuery, "c.trashed_at IS NULL") {
		t.Fatalf("expected trendingCoursesQuery to filter c.trashed_at IS NULL")
	}
	// Draft-only courses are excluded structurally: the INNER JOIN course_versions pv
	// ON pv.id = c.current_published_version_id requires a non-null published version.
	if !strings.Contains(trendingCoursesQuery, "INNER JOIN course_versions pv") {
		t.Fatalf("expected trendingCoursesQuery to INNER JOIN course_versions on the published version (excludes draft-only courses)")
	}
}

// TestTrendingCoursesQueryExplicitlyMatchesPartialIndexPredicate guards
// against a Big-O regression: idx_courses_published_created_at is a PARTIAL
// index (WHERE deleted_at IS NULL AND trashed_at IS NULL AND
// current_published_version_id IS NOT NULL). Postgres can only use a partial
// index when the query's own WHERE clause provably implies the index
// predicate; it does not reliably infer "current_published_version_id IS NOT
// NULL" from the INNER JOIN condition alone. Without this explicit clause,
// the planner may fall back to a full sort of every non-trashed course
// (O(N log N)) instead of an index-driven scan bounded by LIMIT (O(log N + K)).
func TestTrendingCoursesQueryExplicitlyMatchesPartialIndexPredicate(t *testing.T) {
	if !strings.Contains(trendingCoursesQuery, "c.current_published_version_id IS NOT NULL") {
		t.Fatalf("expected trendingCoursesQuery to explicitly filter c.current_published_version_id IS NOT NULL (matches idx_courses_published_created_at's predicate), got:\n%s", trendingCoursesQuery)
	}
}
