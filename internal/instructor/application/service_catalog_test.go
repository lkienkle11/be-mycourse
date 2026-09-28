package application

import "testing"

func TestPopularInstructorsCacheKeyIncludesLimit(t *testing.T) {
	if got, want := popularInstructorsCacheKey(4), "mycourse:catalog:popular_instructors:limit:4"; got != want {
		t.Errorf("popularInstructorsCacheKey(4) = %q, want %q", got, want)
	}
	if popularInstructorsCacheKey(4) == popularInstructorsCacheKey(12) {
		t.Errorf("expected different limits to produce different cache keys")
	}
}
