//go:build linux

package monitor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type MirrorProvider interface {
	Create(spec JobSpec) (ifname string, cleanup func() error, err error)
}

type AttachProvider interface {
	Attach(ifname string, spec JobSpec) (cleanup func() error, err error)
}

type Collector interface {
	Run(ctx context.Context, jobID string, spec JobSpec) (ResultsProvider, error)
}

type ResultsProvider interface {
	Summary() interface{}
}

// Supervisor implements Core interface for API handlers
type Supervisor struct {
	mir MirrorProvider
	att AttachProvider
	col Collector

	maxConcurrent int32
	activeJobs    int32

	mu   sync.RWMutex // protects jobs map and fields of *Job
	jobs map[string]*Job
}

func NewSupervisor(m MirrorProvider, a AttachProvider, c Collector, max int) *Supervisor {
	return &Supervisor{
		mir: m, att: a, col: c,
		maxConcurrent: int32(max),
		jobs:          make(map[string]*Job),
	}
}

func (s *Supervisor) tryReserve() bool {
	for {
		n := atomic.LoadInt32(&s.activeJobs)
		if n >= s.maxConcurrent {
			return false
		}
		if atomic.CompareAndSwapInt32(&s.activeJobs, n, n+1) {
			return true
		}
	}
}
func (s *Supervisor) release() { atomic.AddInt32(&s.activeJobs, -1) }

// API/Core methods
func (s *Supervisor) TryStartJob(req interface{}) (interface{}, int, error) {
	spec := req.(interface{ ToSpec() JobSpec }).ToSpec()
	if !s.tryReserve() {
		return nil, 429, ErrConcurrencyLimit
	}
	id := uuid.NewString()

	j := &Job{
		ID:        id,
		Spec:      spec,
		State:     JobStarting,
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(spec.Duration),
	}

	ifname, mirCleanup, err := s.mir.Create(spec)
	if err != nil {
		s.release()
		return nil, 500, err
	}
	j.IfName = ifname

	attCleanup, err := s.att.Attach(ifname, spec)
	if err != nil {
		_ = mirCleanup()
		s.release()
		return nil, 500, err
	}

	ctx, cancel := context.WithDeadline(context.Background(), j.ExpiresAt)
	j.cancel = cancel

	// store the job under lock
	s.mu.Lock()
	s.jobs[id] = j
	s.mu.Unlock()

	go func() {
		defer s.release()
		defer attCleanup()
		defer mirCleanup()

		// mark running under lock
		s.mu.Lock()
		if jj, ok := s.jobs[id]; ok {
			jj.State = JobRunning
		}
		s.mu.Unlock()

		// run collector (ignore returned provider for now)
		_, _ = s.col.Run(ctx, id, spec)

		<-ctx.Done()

		// mark done under lock
		s.mu.Lock()
		if jj, ok := s.jobs[id]; ok {
			jj.State = JobDone
		}
		s.mu.Unlock()
	}()

	return map[string]interface{}{"job_id": id, "status": "starting", "interface": ifname}, 201, nil
}

func (s *Supervisor) GetJob(id string) (interface{}, int, error) {
	s.mu.RLock()
	j, ok := s.jobs[id]
	if !ok {
		s.mu.RUnlock()
		return nil, 404, ErrJobNotFound
	}
	// copy out the fields we need while holding the read lock
	resp := map[string]interface{}{
		"job_id":     j.ID,
		"status":     j.State,
		"started_at": j.StartedAt,
		"expires_at": j.ExpiresAt,
		"port":       j.Spec.Port,
		"interface":  j.IfName,
	}
	s.mu.RUnlock()
	return resp, 200, nil
}

func (s *Supervisor) StopJob(id string) (interface{}, int, error) {
	s.mu.RLock()
	j, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return nil, 404, ErrJobNotFound
	}
	if j.cancel != nil {
		j.cancel()
	}
	return map[string]interface{}{"job_id": j.ID, "status": "stopped"}, 200, nil
}

func (s *Supervisor) GetResults(id string) (interface{}, int, error) {
	// TODO: wire collector summary
	return map[string]interface{}{
		"window_sec": 0, "packets_total": 0, "bytes_total": 0,
		"errors": map[string]uint64{}, "top_flows": []interface{}{},
		"latency_histogram_ns": map[string]interface{}{"bounds": []uint64{}, "counts": []uint64{}},
		"otel_export":          map[string]interface{}{"exported": true, "endpoint": ""},
	}, 200, nil
}
