package infra

import (
	"strings"
	"testing"
)

// Every owner-bearing course query must project owner_slug next to
// owner_display_name through the shared courseListOwnerColumns fragment: a
// missing column silently scans "" into the shared scan rows.
func TestOwnerBearingCourseQueriesSelectOwnerSlug(t *testing.T) {
	queries := map[string]string{
		"adminCoursePublishedListSelect": adminCoursePublishedListSelect,
		"trendingCoursesQuery":           trendingCoursesQuery,
		"continueLearningQuery":          continueLearningQuery,
		"pendingReviewsListQuery":        pendingReviewsListQuery,
	}
	for name, q := range queries {
		t.Run(name, func(t *testing.T) {
			for _, fragment := range []string{
				"AS owner_display_name",
				"COALESCE(ou.slug, '') AS owner_slug",
			} {
				if !strings.Contains(q, fragment) {
					t.Fatalf("%s must select %q", name, fragment)
				}
			}
		})
	}
}

func TestCollaboratorProfileSelectSQLReusesPickerProjectionWithSlug(t *testing.T) {
	for _, fragment := range []string{
		"AS user_id",
		"AS display_name",
		"AS avatar_url",
		"COALESCE(u.slug, '') AS slug",
		"u.id IN @user_ids",
	} {
		if !strings.Contains(collaboratorProfileSelectSQL, fragment) {
			t.Fatalf("collaboratorProfileSelectSQL must contain %q", fragment)
		}
	}
}
