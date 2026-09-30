package application

import "testing"

func TestTrendingCoursesCacheKeyIncludesLimit(t *testing.T) {
	if got, want := trendingCoursesCacheKey(8), "mycourse:catalog:trending_courses:limit:8"; got != want {
		t.Errorf("trendingCoursesCacheKey(8) = %q, want %q", got, want)
	}
	if trendingCoursesCacheKey(8) == trendingCoursesCacheKey(24) {
		t.Errorf("expected different limits to produce different cache keys")
	}
}
