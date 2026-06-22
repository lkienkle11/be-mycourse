package gormx

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

func TestSQLLogger_trace_logsEveryQueryWithDuration(t *testing.T) {
	var buf bytes.Buffer
	l := newSQLLogger(&buf, false).(*sqlLogger)

	l.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	out := buf.String()
	if !strings.Contains(out, "[50.000ms]") {
		t.Fatalf("expected duration in log, got: %q", out)
	}
	if !strings.Contains(out, "[rows:1]") {
		t.Fatalf("expected rows in log, got: %q", out)
	}
	if !strings.Contains(out, "SELECT 1") {
		t.Fatalf("expected SQL in log, got: %q", out)
	}
}

func TestSQLLogger_trace_colorsByDuration(t *testing.T) {
	cases := []struct {
		name     string
		elapsed  time.Duration
		wantCode string
	}{
		{name: "fast", elapsed: 10 * time.Millisecond, wantCode: gormlogger.Green},
		{name: "warn", elapsed: 500 * time.Millisecond, wantCode: gormlogger.Yellow},
		{name: "slow", elapsed: 3 * time.Second, wantCode: gormlogger.Red},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := newSQLLogger(&buf, true).(*sqlLogger)
			begin := time.Now().Add(-tc.elapsed)

			l.Trace(context.Background(), begin, func() (string, int64) {
				return "SELECT color", 0
			}, nil)

			if !strings.Contains(buf.String(), tc.wantCode+"SELECT color") {
				t.Fatalf("expected %q color around SQL, got: %q", tc.name, buf.String())
			}
		})
	}
}

func TestSQLLogger_trace_ignoresRecordNotFoundWhenConfigured(t *testing.T) {
	var buf bytes.Buffer
	l := newSQLLogger(&buf, false).(*sqlLogger)

	l.Trace(context.Background(), time.Now().Add(-5*time.Millisecond), func() (string, int64) {
		return "SELECT missing", 0
	}, gormlogger.ErrRecordNotFound)

	out := buf.String()
	if strings.Contains(out, "[error]") {
		t.Fatalf("record not found should not be logged as error, got: %q", out)
	}
	if !strings.Contains(out, "SELECT missing") {
		t.Fatalf("expected normal SQL log, got: %q", out)
	}
}

func TestSQLLogger_trace_logsRealErrors(t *testing.T) {
	var buf bytes.Buffer
	l := newSQLLogger(&buf, false).(*sqlLogger)
	err := errors.New("dial timeout")

	l.Trace(context.Background(), time.Now().Add(-2*time.Second), func() (string, int64) {
		return "SELECT slow", -1
	}, err)

	out := buf.String()
	if !strings.Contains(out, "dial timeout") {
		t.Fatalf("expected driver error in log, got: %q", out)
	}
	if !strings.Contains(out, "[2000.000ms]") {
		t.Fatalf("expected duration in error log, got: %q", out)
	}
	if !strings.Contains(out, "[rows:-]") {
		t.Fatalf("expected unknown rows marker, got: %q", out)
	}
}

func TestSQLLogThresholds(t *testing.T) {
	fast, slow := SQLLogThresholds()
	if fast != sqlLogFastThreshold || slow != sqlLogSlowThreshold {
		t.Fatalf("unexpected thresholds: fast=%v slow=%v", fast, slow)
	}
	if !strings.Contains(SQLLogThresholdSummary(), "green") {
		t.Fatalf("expected summary text, got: %q", SQLLogThresholdSummary())
	}
}
