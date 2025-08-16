//go:build linux

package monitor

import (
	"context"
	"time"

	"github.com/platformbuilds/telegen-sonic/pkg/api"
)

// BPFCollectorAdapter satisfies the Supervisor's Collector interface by delegating to MetricsCollector.
type BPFCollectorAdapter struct {
	mc *MetricsCollector
}

// NewBPFCollector is the factory main.go calls.
func NewBPFCollector(mc *MetricsCollector) *BPFCollectorAdapter {
	return &BPFCollectorAdapter{mc: mc}
}

type noopResults struct{}

func (noopResults) Summary() interface{} { return map[string]any{} }

func (a *BPFCollectorAdapter) Run(ctx context.Context, jobID string, spec JobSpec) (ResultsProvider, error) {
	go func() { _ = a.mc.Start(ctx) }()
	return noopResults{}, nil
}

// CoreAdapter translates between the generic Supervisor methods (map[string]any)
// and the typed api.Core interface used by the HTTP layer.
type CoreAdapter struct {
	S *Supervisor
}

func (c *CoreAdapter) TryStartJob(req api.StartJobRequest) (api.StartJobResponse, int, error) {
	resp, code, err := c.S.TryStartJob(req)
	if err != nil {
		return api.StartJobResponse{}, code, err
	}
	m, _ := resp.(map[string]any)
	return api.StartJobResponse{
		JobID:     asString(m, "job_id"),
		Status:    asString(m, "status"),
		Interface: asString(m, "interface"),
	}, code, nil
}

func (c *CoreAdapter) GetJob(id string) (api.JobStatus, int, error) {
	resp, code, err := c.S.GetJob(id)
	if err != nil {
		return api.JobStatus{}, code, err
	}
	m, _ := resp.(map[string]any)
	return api.JobStatus{
		JobID:     asString(m, "job_id"),
		Status:    asString(m, "status"),
		StartedAt: asTime(m, "started_at"),
		ExpiresAt: asTime(m, "expires_at"),
		Port:      asString(m, "port"),
		Interface: asString(m, "interface"),
	}, code, nil
}

func (c *CoreAdapter) StopJob(id string) (api.StopJobResponse, int, error) {
	resp, code, err := c.S.StopJob(id)
	if err != nil {
		return api.StopJobResponse{}, code, err
	}
	m, _ := resp.(map[string]any)
	return api.StopJobResponse{
		JobID:  asString(m, "job_id"),
		Status: asString(m, "status"),
	}, code, nil
}

func (c *CoreAdapter) GetResults(id string) (api.JobResults, int, error) {
	// TODO: map Supervisor's results payload to api.JobResults when you wire it up.
	_, code, err := c.S.GetResults(id)
	if err != nil {
		return api.JobResults{}, code, err
	}
	return api.JobResults{}, code, nil
}

/* ---------- small helpers for safe conversions ---------- */

func asString(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func asInt(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func asTime(m map[string]any, k string) time.Time {
	switch v := m[k].(type) {
	case time.Time:
		return v
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	}
	return time.Time{}
}
