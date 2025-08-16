//go:build linux

package monitor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

/* ---------- fakes ---------- */

type fakeMirror struct {
	ifname string
	err    error
	calls  int32
}

func (f *fakeMirror) Create(spec JobSpec) (string, func() error, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.err != nil {
		return "", func() error { return nil }, f.err
	}
	return f.ifname, func() error { return nil }, nil
}

type fakeAttach struct {
	err   error
	calls int32
}

func (f *fakeAttach) Attach(ifname string, spec JobSpec) (func() error, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.err != nil {
		return func() error { return nil }, f.err
	}
	return func() error { return nil }, nil
}

type fakeCollector struct {
	runErr error
	calls  int32
}

func (f *fakeCollector) Run(ctx context.Context, jobID string, spec JobSpec) (ResultsProvider, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.runErr != nil {
		return nil, f.runErr
	}
	return fakeResults{}, nil
}

type fakeResults struct{}

func (fakeResults) Summary() interface{} { return map[string]any{} }

/* ---------- helpers ---------- */

type startReq struct {
	spec JobSpec
}

func (s startReq) ToSpec() JobSpec { return s.spec }

/* ---------- tests ---------- */

func TestSupervisor_TryStartGetStop_Success(t *testing.T) {
	mir := &fakeMirror{ifname: "mirror0"}
	att := &fakeAttach{}
	col := &fakeCollector{}
	sup := NewSupervisor(mir, att, col, 2)

	// Start a short job
	spec := JobSpec{Port: "Ethernet0", Duration: 500 * time.Millisecond}
	resp, code, err := sup.TryStartJob(startReq{spec: spec})
	if err != nil || code != 201 {
		t.Fatalf("TryStartJob err=%v code=%d", err, code)
	}
	m := resp.(map[string]interface{})
	jobID := m["job_id"].(string)

	// GetJob returns running/starting state
	jres, code, err := sup.GetJob(jobID)
	if err != nil || code != 200 {
		t.Fatalf("GetJob err=%v code=%d", err, code)
	}
	jm := jres.(map[string]interface{})
	if jm["interface"] != "mirror0" {
		t.Fatalf("expected interface mirror0, got %v", jm["interface"])
	}

	// Stop the job explicitly (should cancel the context)
	sresp, code, err := sup.StopJob(jobID)
	if err != nil || code != 200 {
		t.Fatalf("StopJob err=%v code=%d", err, code)
	}
	if sresp.(map[string]interface{})["status"] != "stopped" {
		t.Fatalf("expected stopped status")
	}
}

func TestSupervisor_ConcurrencyLimit(t *testing.T) {
	mir := &fakeMirror{ifname: "mirror0"}
	att := &fakeAttach{}
	col := &fakeCollector{}
	sup := NewSupervisor(mir, att, col, 1) // allow only 1 job concurrently

	spec := JobSpec{Port: "Eth0", Duration: 2 * time.Second}

	// First job should start
	_, code, err := sup.TryStartJob(startReq{spec})
	if err != nil || code != 201 {
		t.Fatalf("TryStartJob #1 err=%v code=%d", err, code)
	}

	// Second should be rejected with 429
	_, code, err = sup.TryStartJob(startReq{spec})
	if code != 429 || err == nil { // your Supervisor returns ErrConcurrencyLimit
		t.Fatalf("expected concurrency limit (429), got code=%d err=%v", code, err)
	}
}

func TestSupervisor_GetJob_NotFound(t *testing.T) {
	mir := &fakeMirror{ifname: "mirror0"}
	att := &fakeAttach{}
	col := &fakeCollector{}
	sup := NewSupervisor(mir, att, col, 2)

	_, code, err := sup.GetJob("does-not-exist")
	if code != 404 || !errors.Is(err, ErrJobNotFound) {
		t.Fatalf("expected 404 ErrJobNotFound, got code=%d err=%v", code, err)
	}
}
