package monitor

import (
	"context"
	"sync"
	"time"
)

type JobSpec struct {
	Port         string
	Direction    string
	SpanMethod   string
	VLAN         *int
	Filters      map[string]interface{}
	SampleRate   int
	Duration     time.Duration
	OTLPExport   bool
	ResultDetail string
}

type JobState string

const (
	JobStarting JobState = "starting"
	JobRunning  JobState = "running"
	JobStopping JobState = "stopping"
	JobDone     JobState = "done"
	JobFailed   JobState = "failed"
)

type Job struct {
	ID        string
	Spec      JobSpec
	State     JobState
	StartedAt time.Time
	ExpiresAt time.Time
	IfName    string

	mu     sync.Mutex
	cancel context.CancelFunc
}
