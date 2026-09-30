package slug

import (
	"crypto/rand"
	"io"
	"strings"
	"unicode/utf8"
)

const randomSuffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// GenerateRandomSuffix returns n random lowercase-ASCII-letter/digit
// characters using crypto/rand (never math/rand — mirrors the existing
// internal/shared/utils.GenerateRandomDigits precedent). It returns an error
// instead of panicking on RNG failure — a web backend must not crash a
// request just because the entropy source had a transient problem — and uses
// rejection sampling instead of a plain `% len(alphabet)` to avoid the small
// modulo bias that `256 % 36 != 0` would otherwise introduce.
func GenerateRandomSuffix(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	const alphabetLen = byte(len(randomSuffixAlphabet)) // 36
	const maxUnbiased = 252                             // largest multiple of 36 that fits in a byte (36*7)
	out := make([]byte, 0, n)
	buf := make([]byte, 1)
	for len(out) < n {
		if _, err := io.ReadFull(rand.Reader, buf); err != nil {
			return "", err
		}
		if buf[0] >= maxUnbiased {
			continue // reject and redraw — keeps the distribution uniform over the 36-letter alphabet
		}
		out = append(out, randomSuffixAlphabet[buf[0]%alphabetLen])
	}
	return string(out), nil
}

const (
	startSuffixLen    = 7
	attemptsPerLength = 7
)

// RetryWithSuffix generates random suffixes of increasing length (7 chars x7
// attempts, then 8x7, 9x7, ... — no upper bound) and calls build with each
// freshly generated suffix. build is responsible for constructing the actual
// candidate string — so it can apply any domain-specific rule such as a
// maximum total length by truncating its own base before concatenating (see
// TruncateForSuffix) — and for reporting whether that candidate is usable.
// This package never concatenates base+suffix itself and has no opinion on
// column length limits, keeping it reusable for any future table.
func RetryWithSuffix(build func(suffix string) (candidate string, accepted bool, err error)) (string, error) {
	for length := startSuffixLen; ; length++ {
		for range attemptsPerLength {
			suffix, err := GenerateRandomSuffix(length)
			if err != nil {
				return "", err
			}
			candidate, ok, err := build(suffix)
			if err != nil {
				return "", err
			}
			if ok {
				return candidate, nil
			}
		}
	}
}

// TruncateToLen trims s to at most maxLen bytes, never splitting a UTF-8
// codepoint, and trims any trailing hyphen the cut may have left (so the
// result still satisfies the "no trailing hyphen" slug format rule after a
// caller re-validates it).
func TruncateToLen(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	cut := s[:maxLen]
	for !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimRight(cut, "-")
}

// TruncateForSuffix trims base so that base + "-" + a suffix of length
// suffixLen fits within maxTotalLen bytes total. Callers pass their own
// column limit (e.g. 255 for courses.slug) — this package has no opinion on
// it, keeping it reusable for a future table with a different limit.
func TruncateForSuffix(base string, maxTotalLen, suffixLen int) string {
	return TruncateToLen(base, maxTotalLen-1-suffixLen) // -1 for the hyphen
}
