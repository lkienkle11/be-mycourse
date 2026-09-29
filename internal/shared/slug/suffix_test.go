package slug

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestGenerateRandomSuffix(t *testing.T) {
	s, err := GenerateRandomSuffix(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 7 {
		t.Fatalf("length = %d, want 7", len(s))
	}
	for _, r := range s {
		if !strings.ContainsRune(randomSuffixAlphabet, r) {
			t.Fatalf("suffix %q contains character %q outside alphabet %q", s, r, randomSuffixAlphabet)
		}
	}
}

func TestGenerateRandomSuffixDiffers(t *testing.T) {
	a, err := GenerateRandomSuffix(12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := GenerateRandomSuffix(12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == b {
		t.Fatalf("two calls produced the same suffix %q — RNG likely not wired correctly", a)
	}
}

func TestGenerateRandomSuffixDistributionIsNotWildlySkewed(t *testing.T) {
	counts := make(map[rune]int, len(randomSuffixAlphabet))
	const totalChars = 36 * 200 // 200 suffixes worth of characters, enough to smooth out noise
	generated := 0
	for generated < totalChars {
		s, err := GenerateRandomSuffix(36)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range s {
			counts[r]++
			generated++
		}
	}
	expectedPerChar := float64(generated) / float64(len(randomSuffixAlphabet))
	for _, r := range randomSuffixAlphabet {
		c := float64(counts[r])
		// Rejection sampling should keep this close to uniform; allow generous
		// slack (3x expected) purely to guard against a gross regression (e.g.
		// reintroducing `% 36` bias or a broken alphabet index), not to assert
		// statistical rigor.
		if c > expectedPerChar*3 {
			t.Fatalf("character %q appears %v times, expected ~%v — possible modulo bias regression", r, c, expectedPerChar)
		}
	}
}

func TestRetryWithSuffixEscalatesLength(t *testing.T) {
	var seenLengths []int
	rejectCount := 0
	_, err := RetryWithSuffix(func(suffix string) (string, bool, error) {
		seenLengths = append(seenLengths, len(suffix))
		rejectCount++
		if rejectCount <= attemptsPerLength { // reject every 7-char attempt
			return "", false, nil
		}
		return "base-" + suffix, true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seenLengths) != attemptsPerLength+1 {
		t.Fatalf("expected %d attempts before acceptance, got %d", attemptsPerLength+1, len(seenLengths))
	}
	for i, l := range seenLengths {
		want := startSuffixLen
		if i == attemptsPerLength {
			want = startSuffixLen + 1
		}
		if l != want {
			t.Fatalf("attempt %d: suffix length = %d, want %d", i, l, want)
		}
	}
}

func TestRetryWithSuffixFirstAttemptAccepted(t *testing.T) {
	calls := 0
	candidate, err := RetryWithSuffix(func(suffix string) (string, bool, error) {
		calls++
		return "base-" + suffix, true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
	if !strings.HasPrefix(candidate, "base-") || len(candidate) != len("base-")+startSuffixLen {
		t.Fatalf("unexpected candidate %q", candidate)
	}
}

func TestTruncateToLenNeverSplitsUTF8AndTrimsHyphen(t *testing.T) {
	// "café-" — é is 2 bytes in UTF-8; cutting at byte 4 would split it.
	s := "café-shop"
	got := TruncateToLen(s, 4)
	if !utf8.ValidString(got) {
		t.Fatalf("TruncateToLen(%q, 4) = %q is not valid UTF-8 (split a codepoint)", s, got)
	}
	if strings.HasSuffix(got, "-") {
		t.Fatalf("TruncateToLen(%q, 4) = %q must not end with a trailing hyphen", s, got)
	}
}

func TestTruncateToLenShorterThanMaxIsUnchanged(t *testing.T) {
	if got := TruncateToLen("abc", 10); got != "abc" {
		t.Fatalf("got %q, want \"abc\"", got)
	}
}

func TestTruncateForSuffixNeverExceedsTotalLength(t *testing.T) {
	base := strings.Repeat("a", 300) // deliberately longer than any realistic maxTotalLen
	const maxTotalLen = 255
	for suffixLen := 7; suffixLen <= 10; suffixLen++ {
		suffix := strings.Repeat("x", suffixLen)
		truncatedBase := TruncateForSuffix(base, maxTotalLen, suffixLen)
		candidate := truncatedBase + "-" + suffix
		if len(candidate) > maxTotalLen {
			t.Fatalf("suffixLen=%d: candidate length %d exceeds maxTotalLen %d (candidate=%q)", suffixLen, len(candidate), maxTotalLen, candidate)
		}
	}
}
