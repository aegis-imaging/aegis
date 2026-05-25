package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/middleware"
)

// TestLoggingPreservesFlusher locks in the SSE-500 regression fix. The
// Logging middleware wraps http.ResponseWriter in its own struct; that
// struct must continue to expose http.Flusher (via its own Flush method)
// or every SSE handler under Logging returns 500 "streaming not
// supported." This test wires Logging around a tiny handler that
// type-asserts to http.Flusher — failure means the assertion no longer
// holds.
func TestLoggingPreservesFlusher(t *testing.T) {
	h := middleware.Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Fatal("wrapped writer must implement http.Flusher; SSE handlers depend on it")
		}
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sse-flush-probe", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("handler should have written 200, got %d", rr.Code)
	}
}
