//go:build linux

package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/platformbuilds/telegen-sonic/pkg/api"
)

// ---- test double for api.Core ----
type testCore struct {
	startCalled, getCalled, stopCalled, resultsCalled bool

	startResp api.StartJobResponse
	startCode int
	startErr  error

	getResp api.JobStatus
	getCode int
	getErr  error

	stopResp api.StopJobResponse
	stopCode int
	stopErr  error

	resultsResp api.JobResults
	resultsCode int
	resultsErr  error
}

func (t *testCore) TryStartJob(req api.StartJobRequest) (api.StartJobResponse, int, error) {
	t.startCalled = true
	if t.startCode == 0 {
		t.startCode = http.StatusCreated
	}
	return t.startResp, t.startCode, t.startErr
}
func (t *testCore) GetJob(id string) (api.JobStatus, int, error) {
	t.getCalled = true
	if t.getCode == 0 {
		t.getCode = http.StatusOK
	}
	return t.getResp, t.getCode, t.getErr
}
func (t *testCore) StopJob(id string) (api.StopJobResponse, int, error) {
	t.stopCalled = true
	if t.stopCode == 0 {
		t.stopCode = http.StatusOK
	}
	return t.stopResp, t.stopCode, t.stopErr
}
func (t *testCore) GetResults(id string) (api.JobResults, int, error) {
	t.resultsCalled = true
	if t.resultsCode == 0 {
		t.resultsCode = http.StatusOK
	}
	return t.resultsResp, t.resultsCode, t.resultsErr
}

// ---- tests ----

func TestAgent_Router_Success(t *testing.T) {
	core := &testCore{
		startResp: api.StartJobResponse{JobID: "j1", Status: "started", Interface: "eth0"},
		getResp: api.JobStatus{
			JobID:     "j1",
			Status:    "running",
			StartedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Minute),
			Port:      "Ethernet0",
			Interface: "eth0",
		},
		stopResp:    api.StopJobResponse{JobID: "j1", Status: "stopped"},
		resultsResp: api.JobResults{WindowSec: 60, Packets: 100, Bytes: 200},
	}
	h := &api.Handlers{Core: core}
	srv := httptest.NewServer(api.NewRouter(h))
	defer srv.Close()

	// POST /v1/monitor/jobs
	resp, err := http.Post(srv.URL+"/v1/monitor/jobs", "application/json",
		bytes.NewBufferString(`{"port":"Ethernet0","direction":"ingress","duration_sec":5}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StartJob status=%d", resp.StatusCode)
	}
	// GET /v1/monitor/jobs/j1
	resp, _ = http.Get(srv.URL + "/v1/monitor/jobs/j1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GetJob status=%d", resp.StatusCode)
	}
	// DELETE /v1/monitor/jobs/j1
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/v1/monitor/jobs/j1", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StopJob status=%d", resp.StatusCode)
	}
	// GET /v1/monitor/jobs/j1/results
	resp, _ = http.Get(srv.URL + "/v1/monitor/jobs/j1/results")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GetResults status=%d", resp.StatusCode)
	}

	// sanity: calls happened
	if !core.startCalled || !core.getCalled || !core.stopCalled || !core.resultsCalled {
		t.Fatalf("core methods not all called: %+v", core)
	}
}

func TestAgent_Router_Errors(t *testing.T) {
	core := &testCore{
		startCode:   http.StatusBadRequest,
		startErr:    errors.New("bad start"),
		getCode:     http.StatusNotFound,
		getErr:      errors.New("not found"),
		stopCode:    http.StatusConflict,
		stopErr:     errors.New("conflict"),
		resultsCode: http.StatusInternalServerError,
		resultsErr:  errors.New("boom"),
	}
	h := &api.Handlers{Core: core}
	srv := httptest.NewServer(api.NewRouter(h))
	defer srv.Close()

	tests := []struct {
		method string
		path   string
		body   string
		code   int
	}{
		{http.MethodPost, "/v1/monitor/jobs", `{}`, http.StatusBadRequest},
		{http.MethodGet, "/v1/monitor/jobs/nope", ``, http.StatusNotFound},
		{http.MethodDelete, "/v1/monitor/jobs/nope", ``, http.StatusConflict},
		{http.MethodGet, "/v1/monitor/jobs/nope/results", ``, http.StatusInternalServerError},
	}
	for _, tc := range tests {
		var req *http.Request
		if tc.method == http.MethodPost {
			req, _ = http.NewRequest(tc.method, srv.URL+tc.path, bytes.NewBufferString(tc.body))
		} else {
			req, _ = http.NewRequest(tc.method, srv.URL+tc.path, nil)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		if resp.StatusCode != tc.code {
			t.Fatalf("%s %s got %d want %d", tc.method, tc.path, resp.StatusCode, tc.code)
		}
	}
}

// Tiny check to keep the version vars covered in this package.
func TestAgent_VersionVars(t *testing.T) {
	if version == "" || commit == "" {
		t.Fatalf("version/commit should be set (got %q %q)", version, commit)
	}
}
