// Package timex provides Unix-epoch-second helpers for audit timestamps.
package timex

import "time"

// NowUnix returns the current time as Unix epoch seconds.
func NowUnix() int64 {
	return time.Now().Unix()
}

// PtrUnix returns a pointer to t (for nullable deleted_at columns).
func PtrUnix(t int64) *int64 {
	return &t
}

// UnixOrZero returns *p when non-nil, otherwise 0.
func UnixOrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
