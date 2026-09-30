package infra

import (
	"strings"
	"testing"
)

func TestContinueLearningQueryOrdersByRecentActivityFallingBackToEnrollment(t *testing.T) {
	if !strings.Contains(continueLearningQuery, "ORDER BY COALESCE(prog.last_interacted_at, e.created_at) DESC") {
		t.Fatalf("expected continueLearningQuery to order by recent activity falling back to enrollment time, got:\n%s", continueLearningQuery)
	}
}

// TestContinueLearningQueryCountsAllSubLessonKinds guards against
// re-narrowing progress to VIDEO-only sub-lessons: a course's outline
// legitimately mixes VIDEO/QUIZ/TEXT sub-lessons under one course, so
// progress must count every sub-lesson kind (product decision).
func TestContinueLearningQueryCountsAllSubLessonKinds(t *testing.T) {
	if strings.Contains(continueLearningQuery, "kind = 'VIDEO'") {
		t.Fatalf("continueLearningQuery must not filter by sub-lesson kind — it must count all kinds (VIDEO/QUIZ/TEXT), got:\n%s", continueLearningQuery)
	}
}

// TestContinueLearningQueryDerivesCompletionFromOutlineNotContentType guards
// against trusting the client-supplied, unvalidated
// course_progress_items.content_type column — completion must be derived
// from course_sub_lessons (the authoritative outline) joined by stable_id
// instead (see design.md Risks).
func TestContinueLearningQueryDerivesCompletionFromOutlineNotContentType(t *testing.T) {
	if !strings.Contains(continueLearningQuery, "FROM course_sub_lessons sl") {
		t.Fatalf("expected continueLearningQuery to join from course_sub_lessons (the authoritative outline), got:\n%s", continueLearningQuery)
	}
	if strings.Contains(continueLearningQuery, "content_type = 'VIDEO'") {
		t.Fatalf("continueLearningQuery must not trust the client-supplied course_progress_items.content_type column")
	}
}

func TestContinueLearningQueryScopesToOwnEnrollments(t *testing.T) {
	if !strings.Contains(continueLearningQuery, "WHERE e.user_id = ? AND e.deleted_at IS NULL") {
		t.Fatalf("expected continueLearningQuery to filter WHERE e.user_id = ? AND e.deleted_at IS NULL, got:\n%s", continueLearningQuery)
	}
}

// TestContinueLearningQueryLastActivityIsPerEnrollmentLateral guards against a
// real complexity bug found by manual Big-O review: an earlier draft joined a
// plain (non-LATERAL) "SELECT ... FROM course_progress_items GROUP BY
// enrollment_id" derived table on enrollment_id = e.id. Postgres cannot push
// e.id into a GROUP BY subquery before aggregating it, so that plan would
// aggregate every course_progress_items row for every enrollment on the
// entire platform (O(P), P = platform-wide progress-row count) just to read
// one learner's last-activity timestamp. The subquery must be LATERAL and
// filter by enrollment_id = e.id INSIDE itself, so it is bounded by this
// learner's own row count instead (uses uix_course_progress_items_active,
// which leads with enrollment_id).
func TestContinueLearningQueryLastActivityIsPerEnrollmentLateral(t *testing.T) {
	if !strings.Contains(continueLearningQuery, "LEFT JOIN LATERAL (\n    SELECT MAX(last_interacted_at) AS last_interacted_at\n    FROM course_progress_items\n    WHERE enrollment_id = e.id AND deleted_at IS NULL\n) prog ON TRUE") {
		t.Fatalf("expected the last-activity subquery to be LATERAL and filtered by enrollment_id = e.id, got:\n%s", continueLearningQuery)
	}
}
