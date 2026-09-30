package infra

import (
	"strings"
	"testing"

	"mycourse-io-be/internal/shared/userpicker"
)

func TestPopularInstructorsQueryOrdersByCourseCountThenRecency(t *testing.T) {
	if !strings.Contains(popularInstructorsQuery, "ORDER BY agg.course_count DESC, agg.latest_course_at DESC") {
		t.Fatalf("expected popularInstructorsQuery to order by course_count desc then recency, got:\n%s", popularInstructorsQuery)
	}
}

func TestPopularInstructorsQueryFiltersByInstructorRole(t *testing.T) {
	if !strings.Contains(popularInstructorsQuery, "r.name = @role_name") {
		t.Fatalf("expected popularInstructorsQuery to filter on r.name = @role_name, got:\n%s", popularInstructorsQuery)
	}
}

// TestPopularInstructorsQueryReusesActiveUserWhereClause guards against
// hand-rolling a weaker is_disable-only check that would miss actively-banned
// accounts — must reuse the existing userpicker.ActiveUserWhereClause() verbatim.
func TestPopularInstructorsQueryReusesActiveUserWhereClause(t *testing.T) {
	if !strings.Contains(popularInstructorsQuery, userpicker.ActiveUserWhereClause()) {
		t.Fatalf("expected popularInstructorsQuery to embed userpicker.ActiveUserWhereClause() verbatim, got:\n%s", popularInstructorsQuery)
	}
}

func TestPopularInstructorsQueryExcludesUnpublishedAndTrashedCourses(t *testing.T) {
	if !strings.Contains(popularInstructorsQuery, "c.current_published_version_id IS NOT NULL") {
		t.Fatalf("expected popularInstructorsQuery to only count published courses")
	}
	if !strings.Contains(popularInstructorsQuery, "c.trashed_at IS NULL") {
		t.Fatalf("expected popularInstructorsQuery to exclude trashed courses")
	}
}
