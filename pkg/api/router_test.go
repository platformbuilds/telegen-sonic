package api

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRouter_SuccessPaths(t *testing.T) {
	tc := &testCore{
		tryStartResp: StartJobResponse{JobID: "j123", Status: "started", Interface: "eth0"},
		tryStartCode: http.StatusCreated,
		getJobResp: JobStatus{
			JobID:     "j123",
			Status:    "running",
			StartedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Port:      "Ethernet0",
			Interface: "eth0",
		},
		stopResp: StopJobResponse{JobID: "j123", Status: "stopped"},
		resultsResp: JobResults{
			WindowSec: 60,
			Packets:   1000,
			Bytes:     2048,
		},
	}
	h := &Handlers{Core: tc}
	srv := httptest.NewServer(NewRouter(h))
	defer srv.Close()

	// StartJob
	body := `{"port":"Ethernet0","direction":"ingress","sample_rate":1}`
	resp, err := http.Post(srv.URL+"/v1/monitor/jobs", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StartJob wrong status: %d", resp.StatusCode)
	}
	if !tc.startCalled {
		t.Fatalf("expected TryStartJob called")
	}

	// GetJob
	resp, _ = http.Get(srv.URL + "/v1/monitor/jobs/j123")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GetJob wrong status: %d", resp.StatusCode)
	}
	if !tc.getCalled {
		t.Fatalf("expected GetJob called")
	}

	// StopJob
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/v1/monitor/jobs/j123", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StopJob wrong status: %d", resp.StatusCode)
	}
	if !tc.stopCalled {
		t.Fatalf("expected StopJob called")
	}

	// GetResults
	resp, _ = http.Get(srv.URL + "/v1/monitor/jobs/j123/results")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GetResults wrong status: %d", resp.StatusCode)
	}
	if !tc.resultsCalled {
		t.Fatalf("expected GetResults called")
	}
}

func TestRouter_ErrorPaths(t *testing.T) {
	tc := &testCore{
		tryStartCode: http.StatusBadRequest,
		tryStartErr:  errors.New("fail-start"),
		getJobCode:   http.StatusNotFound,
		getJobErr:    errors.New("fail-get"),
		stopCode:     http.StatusConflict,
		stopErr:      errors.New("fail-stop"),
		resultsCode:  http.StatusInternalServerError,
		resultsErr:   errors.New("fail-results"),
	}
	h := &Handlers{Core: tc}
	srv := httptest.NewServer(NewRouter(h))
	defer srv.Close()

	tests := []struct {
		method string
		path   string
		code   int
	}{
		{http.MethodPost, "/v1/monitor/jobs", http.StatusBadRequest},
		{http.MethodGet, "/v1/monitor/jobs/xx", http.StatusNotFound},
		{http.MethodDelete, "/v1/monitor/jobs/xx", http.StatusConflict},
		{http.MethodGet, "/v1/monitor/jobs/xx/results", http.StatusInternalServerError},
	}

	for _, tc := range tests {
		var req *http.Request
		if tc.method == http.MethodPost {
			req, _ = http.NewRequest(tc.method, srv.URL+tc.path, bytes.NewBufferString(`{}`))
		} else {
			req, _ = http.NewRequest(tc.method, srv.URL+tc.path, nil)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		if resp.StatusCode != tc.code {
			t.Fatalf("%s %s: got %d, want %d", tc.method, tc.path, resp.StatusCode, tc.code)
		}
	}
}
