package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

/* ------------------ helpers ------------------ */

func makeReqWithRouteParam(method, url, key, val string, body []byte) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func decodeBody[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("json decode failed: %v; body=%s", err, rr.Body.String())
	}
	return out
}

/* ------------------ tests ------------------ */

func TestStartJob_OK(t *testing.T) {
	tc := &testCore{
		tryStartResp: StartJobResponse{JobID: "j1", Status: "starting", Interface: "erspan0"},
		tryStartCode: http.StatusCreated,
	}
	h := &Handlers{Core: tc}

	body := []byte(`{"port":"Ethernet0","direction":"ingress","duration_sec":5}`)
	req := httptest.NewRequest(http.MethodPost, "/jobs/start", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.StartJob(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("code=%d want=%d; body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	got := decodeBody[StartJobResponse](t, rr)
	if got.JobID != "j1" || got.Status != "starting" || got.Interface != "erspan0" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if !tc.startCalled {
		t.Fatalf("expected TryStartJob to be called")
	}
}

func TestStartJob_BadJSON(t *testing.T) {
	h := &Handlers{Core: &testCore{}}
	req := httptest.NewRequest(http.MethodPost, "/jobs/start", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()

	h.StartJob(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want=%d", rr.Code, http.StatusBadRequest)
	}
	var got map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["error"] != "bad_request" {
		t.Fatalf("unexpected error field: %v", got)
	}
}

func TestGetJob_OK(t *testing.T) {
	now := time.Now().UTC()
	tc := &testCore{
		getJobResp: JobStatus{
			JobID:     "j1",
			Status:    "running",
			StartedAt: now,
			ExpiresAt: now.Add(5 * time.Minute),
			Port:      "Ethernet0",
			Interface: "erspan0",
		},
		getJobCode: http.StatusOK,
	}
	h := &Handlers{Core: tc}

	req := makeReqWithRouteParam(http.MethodGet, "/jobs/j1", "job_id", "j1", nil)
	rr := httptest.NewRecorder()

	h.GetJob(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d", rr.Code, http.StatusOK)
	}
	got := decodeBody[JobStatus](t, rr)
	if got.JobID != "j1" || got.Interface != "erspan0" || got.Port != "Ethernet0" || got.Status != "running" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if !tc.getCalled {
		t.Fatalf("expected GetJob called")
	}
}

func TestGetJob_NotFound(t *testing.T) {
	tc := &testCore{
		getJobCode: http.StatusNotFound,
		getJobErr:  errors.New("not found"),
	}
	h := &Handlers{Core: tc}

	req := makeReqWithRouteParam(http.MethodGet, "/jobs/none", "job_id", "none", nil)
	rr := httptest.NewRecorder()

	h.GetJob(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code=%d want=%d", rr.Code, http.StatusNotFound)
	}
	var got map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["error"] != "not_found" {
		t.Fatalf("unexpected error field: %v", got)
	}
}

func TestStopJob_OK(t *testing.T) {
	tc := &testCore{
		stopResp: StopJobResponse{JobID: "j1", Status: "stopped"},
		stopCode: http.StatusOK,
	}
	h := &Handlers{Core: tc}

	req := makeReqWithRouteParam(http.MethodDelete, "/jobs/j1/stop", "job_id", "j1", nil)
	rr := httptest.NewRecorder()

	h.StopJob(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d", rr.Code, http.StatusOK)
	}
	got := decodeBody[StopJobResponse](t, rr)
	if got.JobID != "j1" || got.Status != "stopped" {
		t.Fatalf("unexpected response: %+v", got)
	}
	if !tc.stopCalled {
		t.Fatalf("expected StopJob called")
	}
}

func TestGetResults_OK(t *testing.T) {
	tc := &testCore{
		resultsResp: JobResults{
			WindowSec: 10,
			Packets:   123,
			Bytes:     456,
		},
		resultsCode: http.StatusOK,
	}
	h := &Handlers{Core: tc}

	req := makeReqWithRouteParam(http.MethodGet, "/jobs/j1/results", "job_id", "j1", nil)
	rr := httptest.NewRecorder()

	h.GetResults(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d want=%d", rr.Code, http.StatusOK)
	}
	got := decodeBody[JobResults](t, rr)
	if got.WindowSec != 10 || got.Packets != 123 || got.Bytes != 456 {
		t.Fatalf("unexpected response: %+v", got)
	}
	if !tc.resultsCalled {
		t.Fatalf("expected GetResults called")
	}
}

func TestGetResults_Error(t *testing.T) {
	tc := &testCore{
		resultsCode: http.StatusInternalServerError,
		resultsErr:  errors.New("boom"),
	}
	h := &Handlers{Core: tc}

	req := makeReqWithRouteParam(http.MethodGet, "/jobs/j1/results", "job_id", "j1", nil)
	rr := httptest.NewRecorder()

	h.GetResults(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("code=%d want=%d", rr.Code, http.StatusInternalServerError)
	}
	var got map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["error"] != "results_failed" {
		t.Fatalf("unexpected error field: %v", got)
	}
}
