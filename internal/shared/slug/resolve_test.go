package slug

import (
	"errors"
	"strings"
	"testing"
)

// takenSet builds an AvailableFunc where every listed candidate is taken.
func takenSet(taken ...string) AvailableFunc {
	set := make(map[string]struct{}, len(taken))
	for _, s := range taken {
		set[s] = struct{}{}
	}
	return func(c string) (bool, error) {
		_, hit := set[c]
		return !hit, nil
	}
}

func TestResolveCreateManualFreeIsUsedAsIs(t *testing.T) {
	got, conflict, err := ResolveCreate("golang-course", "ignored title", KindCourse, 255, takenSet())
	if err != nil || conflict != nil || got != "golang-course" {
		t.Fatalf("got (%q, %v, %v)", got, conflict, err)
	}
}

func TestResolveCreateManualTakenReturnsFreeRecommendation(t *testing.T) {
	got, conflict, err := ResolveCreate("golang-course", "", KindCourse, 255, takenSet("golang-course"))
	if err != nil || got != "" {
		t.Fatalf("got (%q, %v)", got, err)
	}
	if conflict == nil || !strings.HasPrefix(conflict.RecommendedSlug, "golang-course-") {
		t.Fatalf("expected suffixed recommendation, got %+v", conflict)
	}
	if len(conflict.RecommendedSlug) != len("golang-course-")+startSuffixLen {
		t.Fatalf("unexpected suffix length in %q", conflict.RecommendedSlug)
	}
}

func TestResolveCreateAutoFreeBaseIsUsed(t *testing.T) {
	got, conflict, err := ResolveCreate("", "Lập trình Golang cơ bản", KindCourse, 255, takenSet())
	if err != nil || conflict != nil || got != "lap-trinh-golang-co-ban" {
		t.Fatalf("got (%q, %v, %v)", got, conflict, err)
	}
}

func TestResolveCreateAutoTakenBaseIsSuffixed(t *testing.T) {
	got, conflict, err := ResolveCreate("", "Golang", KindCourse, 255, takenSet("golang"))
	if err != nil || conflict != nil {
		t.Fatalf("got (%q, %v, %v)", got, conflict, err)
	}
	if !strings.HasPrefix(got, "golang-") || got == "golang" {
		t.Fatalf("expected suffixed base, got %q", got)
	}
}

func TestResolveCreateUnusableNameFallsBackToKindWithSuffix(t *testing.T) {
	got, _, err := ResolveCreate("", "🔥🔥", KindUser, 255, takenSet())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "user-") || len(got) != len("user-")+startSuffixLen {
		t.Fatalf("expected user-<7 chars>, got %q", got)
	}
}

func TestResolveCreateReportsAvailabilityError(t *testing.T) {
	boom := errors.New("db down")
	avail := func(string) (bool, error) { return false, boom }
	if _, _, err := ResolveCreate("x", "", KindTag, 255, avail); !errors.Is(err, boom) {
		t.Fatalf("manual: want boom, got %v", err)
	}
	if _, _, err := ResolveCreate("", "name", KindTag, 255, avail); !errors.Is(err, boom) {
		t.Fatalf("auto: want boom, got %v", err)
	}
}

func TestResolveUpdateFreeAndTaken(t *testing.T) {
	got, err := ResolveUpdate("new-slug", 255, takenSet())
	if err != nil || got != "new-slug" {
		t.Fatalf("free: got (%q, %v)", got, err)
	}
	got, err = ResolveUpdate("new-slug", 255, takenSet("new-slug"))
	if err != nil || !strings.HasPrefix(got, "new-slug-") {
		t.Fatalf("taken: got (%q, %v)", got, err)
	}
}

func TestResolveRetriesEscalatingSuffixUntilFree(t *testing.T) {
	calls := 0
	avail := func(c string) (bool, error) {
		calls++
		if c == "base" {
			return false, nil
		}
		return calls > 10, nil // first suffixed candidates are taken
	}
	got, err := ResolveUpdate("base", 255, avail)
	if err != nil || !strings.HasPrefix(got, "base-") {
		t.Fatalf("got (%q, %v)", got, err)
	}
	if calls <= 2 {
		t.Fatalf("expected several attempts, got %d calls", calls)
	}
}

func TestResolveErrorsWhenSuffixLeavesNoRoom(t *testing.T) {
	// maxLen shorter than hyphen plus 7-char suffix: no base can survive.
	if _, err := ResolveUpdate("abc", 5, takenSet("abc")); err == nil {
		t.Fatal("expected an error when no room remains for the suffix")
	}
}
