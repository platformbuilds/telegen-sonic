//go:build linux

package api

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware_PassesThroughAndLogs(t *testing.T) {
	// Capture logs
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	// A simple next handler that returns 204
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := &Handlers{} // Core not needed for this middleware test
	mw := h.LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	// Ensure the request passed through to next
	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d, want %d", rr.Code, http.StatusNoContent)
	}

	// Ensure something meaningful was logged
	logged := buf.String()
	if !strings.Contains(logged, "GET") || !strings.Contains(logged, "/test/path") {
		t.Fatalf("expected log to contain method and path; got: %q", logged)
	}

	// (Optionally) check that a duration was logged (best-effort)
	if !strings.Contains(logged, "ms") && !strings.Contains(logged, "µs") && !strings.Contains(logged, "ns") && !strings.Contains(logged, "s") {
		t.Logf("log did not contain a duration token; got: %q", logged)
	}
}
