package utils

import (
	"crypto/rand"
	"io"
)

// GenerateRandomDigits returns n decimal digits using crypto/rand.
func GenerateRandomDigits(n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		panic("GenerateRandomDigits: " + err.Error())
	}
	out := make([]byte, n)
	for i := range buf {
		out[i] = '0' + (buf[i] % 10)
	}
	return string(out)
}
