//go:build linux

package monitor

import (
	"testing"
	"time"

	"github.com/cilium/ebpf"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// We purposely DO NOT call c.collectOnce / Start() here because those require
// a real *ebpf.Map with kernel support. These tests validate the parts that
// are independent of the kernel (ctor & helpers).

func TestNewMetricsCollector_Valid(t *testing.T) {
	mp := sdkmetric.NewMeterProvider()
	meter := mp.Meter("test")

	// Pass non-nil *ebpf.Map placeholders; the ctor only checks for nil
	statsMap := &ebpf.Map{}
	ifStatsMap := &ebpf.Map{}

	c, err := NewMetricsCollector(meter, statsMap, ifStatsMap, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatalf("collector is nil")
	}
	if c.interval != 2*time.Second {
		t.Fatalf("interval not respected: %v", c.interval)
	}
}

func TestNewMetricsCollector_Errors(t *testing.T) {
	statsMap := &ebpf.Map{}

	// nil meter
	if _, err := NewMetricsCollector(nil, statsMap, nil, time.Second); err == nil {
		t.Fatalf("expected error for nil meter")
	}

	// nil statsMap
	mp := sdkmetric.NewMeterProvider()
	meter := mp.Meter("test")
	if _, err := NewMetricsCollector(meter, nil, nil, time.Second); err == nil {
		t.Fatalf("expected error for nil statsMap")
	}

	// non-positive interval -> defaulted (no error)
	if c, err := NewMetricsCollector(meter, statsMap, nil, 0); err != nil {
		t.Fatalf("unexpected error for 0 interval: %v", err)
	} else if c.interval <= 0 {
		t.Fatalf("interval was not defaulted")
	}
}

func TestProtoName(t *testing.T) {
	tests := []struct {
		idx  uint32
		want string
	}{
		{idxIPv4, "ipv4"},
		{idxIPv6, "ipv6"},
		{idxICMP6, "icmp6"},
		{idxOther, "other"},
		{99, "other"},
	}
	for _, tc := range tests {
		if got := protoName(tc.idx); got != tc.want {
			t.Fatalf("protoName(%d) = %q, want %q", tc.idx, got, tc.want)
		}
	}
}

func TestDiffU64(t *testing.T) {
	if got := diffU64(10, 3); got != 7 {
		t.Fatalf("diffU64(10,3) = %d, want 7", got)
	}
	// wrap-around handling (current less than previous)
	if got := diffU64(2, 5); got != 2 {
		t.Fatalf("diffU64(2,5) = %d, want 2 (wrap basic handling)", got)
	}
}

func TestSumSliceProtoStats(t *testing.T) {
	in := []ProtoStats{
		{Packets: 1, Bytes: 10},
		{Packets: 2, Bytes: 20},
		{Packets: 3, Bytes: 30},
	}
	out := sumSlice[ProtoStats](in)
	if out.Packets != 6 || out.Bytes != 60 {
		t.Fatalf("sumSlice = %+v, want Packets=6 Bytes=60", out)
	}
}
