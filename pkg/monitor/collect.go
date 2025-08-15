package monitor

import (
	"context"
	"sync"
	"time"
	"fmt"
)

type CollectorImpl struct{
	mu sync.Mutex
	jobs map[string]*aggState
	exportInterval time.Duration
}

type aggState struct {
	packets uint64
	bytes   uint64
	errors  map[string]uint64
	bounds []uint64
	counts []uint64
	start time.Time
	end   time.Time
}

func NewCollector() *CollectorImpl {
	return &CollectorImpl{
		jobs: make(map[string]*aggState),
		exportInterval: 10 * time.Second,
	}
}

func (c *CollectorImpl) Run(ctx context.Context, jobID string, spec JobSpec) (ResultsProvider, error) {
	c.mu.Lock()
	st := &aggState{
		errors: map[string]uint64{},
		bounds: []uint64{10_000, 50_000, 100_000, 500_000, 1_000_000},
		counts: make([]uint64, 5),
		start: time.Now(),
		end:   time.Now().Add(spec.Duration),
	}
	c.jobs[jobID] = st
	c.mu.Unlock()

	go c.exportLoop(ctx, jobID, spec)

	return resultsProvider{c: c, id: jobID}, nil
}

func (c *CollectorImpl) exportLoop(ctx context.Context, jobID string, spec JobSpec) {
	t := time.NewTicker(c.exportInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			c.mu.Lock(); st := c.jobs[jobID]; if st != nil { st.end = time.Now() }; c.mu.Unlock()
			fmt.Println("collector exit for job", jobID)
			return
		case <-t.C:
			// TODO: read BPF maps here and update st; export via OTel in otel.go
			// placeholder increments:
			c.mu.Lock()
			st := c.jobs[jobID]
			if st != nil {
				st.packets += 1000
				st.bytes += 800000
				st.counts[2] += 100 // bump 100us bucket
			}
			c.mu.Unlock()
		}
	}
}

type resultsProvider struct{
	c *CollectorImpl
	id string
}

func (rp resultsProvider) Summary() interface{} {
	rp.c.mu.Lock(); defer rp.c.mu.Unlock()
	st := rp.c.jobs[rp.id]
	if st == nil { return map[string]any{} }
	return map[string]any{
		"window_sec": int(st.end.Sub(st.start).Seconds()),
		"packets_total": st.packets,
		"bytes_total": st.bytes,
		"errors": st.errors,
		"latency_histogram_ns": map[string]any{"bounds": st.bounds, "counts": st.counts},
	}
}
