package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"mycourse-io-be/internal/shared/middleware"
)

func setupRouterWithRequestLogger() (*gin.Engine, *observer.ObservedLogs) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zapcore.DebugLevel)
	base := zap.New(core)
	zap.ReplaceGlobals(base)

	router := gin.New()
	router.Use(middleware.RequestLogger())
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.GET("/bad", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad")
	})
	router.GET("/err", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "err")
	})
	return router, observed
}

func TestRequestLogger_LevelsByStatus(t *testing.T) {
	router, observed := setupRouterWithRequestLogger()

	reqOK := httptest.NewRequest(http.MethodGet, "/ok", nil)
	reqOK.Header.Set("User-Agent", "test-agent")
	wOK := httptest.NewRecorder()
	router.ServeHTTP(wOK, reqOK)

	reqBad := httptest.NewRequest(http.MethodGet, "/bad", nil)
	wBad := httptest.NewRecorder()
	router.ServeHTTP(wBad, reqBad)

	reqErr := httptest.NewRequest(http.MethodGet, "/err", nil)
	wErr := httptest.NewRecorder()
	router.ServeHTTP(wErr, reqErr)

	all := observed.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 access log entries, got %d", len(all))
	}
	if all[0].Level != zap.InfoLevel {
		t.Fatalf("expected info level for 200, got %s", all[0].Level)
	}
	if all[1].Level != zap.WarnLevel {
		t.Fatalf("expected warn level for 400, got %s", all[1].Level)
	}
	if all[2].Level != zap.ErrorLevel {
		t.Fatalf("expected error level for 500, got %s", all[2].Level)
	}

	for _, entry := range all {
		fields := fieldMap(entry.Context)
		if fields["kind"] != "access" {
			t.Fatalf("expected kind=access, got %v", fields["kind"])
		}
		if fields["route"] == "" {
			t.Fatalf("expected non-empty route")
		}
		if fields["latency_ms"] == nil {
			t.Fatalf("expected latency_ms field")
		}
		if fields["bytes"] == nil {
			t.Fatalf("expected bytes field")
		}
		if fields["response_bytes"] == nil {
			t.Fatalf("expected response_bytes alias field")
		}
		if fields["request_id"] == "" {
			t.Fatalf("expected request_id field")
		}
	}
}

func TestRequestLogger_UnmatchedRoute(t *testing.T) {
	router, observed := setupRouterWithRequestLogger()
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	last := observed.All()
	if len(last) != 1 {
		t.Fatalf("expected one access log for unmatched route, got %d", len(last))
	}
	fields := fieldMap(last[0].Context)
	if fields["route"] != "unmatched" {
		t.Fatalf("expected route=unmatched, got %v", fields["route"])
	}
}

func fieldMap(fields []zapcore.Field) map[string]any {
	out := make(map[string]any, len(fields))
	for _, f := range fields {
		switch f.Type {
		case zapcore.StringType:
			out[f.Key] = f.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			out[f.Key] = f.Integer
		case zapcore.DurationType:
			out[f.Key] = f.Integer
		default:
			if f.Interface != nil {
				out[f.Key] = f.Interface
			} else if f.String != "" {
				out[f.Key] = f.String
			} else {
				out[f.Key] = f.Integer
			}
		}
	}
	return out
}
