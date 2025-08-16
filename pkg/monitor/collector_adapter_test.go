//go:build linux

package monitor

import (
	"context"
	"testing"
	"time"
)

func TestBPFCollectorAdapter_Run_NoOp(t *testing.T) {
	// Minimal ctor for MetricsCollector is private; we just ensure adapter exists and Run returns.
	// Using nil receiver is fine because Run is a no-op in this design.
	adapter := NewBPFCollector(nil)

	rp, err := adapter.Run(context.Background(), "job-1", JobSpec{Port: "Eth0", Duration: 10 * time.Second})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if rp == nil || rp.Summary() == nil {
		t.Fatalf("expected non-nil results provider")
	}
}
