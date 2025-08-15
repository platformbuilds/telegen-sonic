package monitor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// BPF map names
const (
	mapStats  = "stats"
	mapProto  = "proto"
)

type CollectorImpl struct{
	m meter
}

type meter interface {
	Counter(name string) (metric.Float64Counter, error)
	Histogram(name string) (metric.Float64Histogram, error)
}

func (c *CollectorImpl) Run(ctx context.Context, jobID string, spec JobSpec) (ResultsProvider, error) {
	m := otel.GetMeterProvider().Meter("sonic-dpmon")
	pktCtr, _ := m.Float64Counter("network.packets")
	byteCtr, _ := m.Float64Counter("network.bytes")

	// Open pinned or in-object maps; here we load from the ELF pinned by tc (tc pinning varies).
	// Pragmatically, open by name using ebpf.LoadPinnedMap if pinned; since tc didn't pin, we open via obj file is non-trivial.
	// Instead, rely on reading via /sys/fs/bpf/tc/globals/<map> (tc creates 'tc/globals' namespace).
	statsMap, err := ebpf.LoadPinnedMap("/sys/fs/bpf/tc/globals/"+mapStats, nil)
	if err != nil {
		log.Printf("warn: open stats map: %v", err)
	}
	protoMap, err := ebpf.LoadPinnedMap("/sys/fs/bpf/tc/globals/"+mapProto, nil)
	if err != nil {
		log.Printf("warn: open proto map: %v", err)
	}

	var lastPkts, lastBytes uint64

	go func(){
		ticker := time.NewTicker(10*time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var pkts, bytes uint64
				if statsMap != nil {
					var key uint32 = 0
					type dpCounters struct{ Packets uint64; Bytes uint64 }
					// Combine per-CPU values
					var agg dpCounters
					iters, _ := statsMap.Lookup(key)
					_ = iters // cilium/ebpf doesn't expose per-cpu read via Lookup; use LookupPerCPU on v0.16
					var perCPU []dpCounters
					if err := statsMap.Lookup(key, &perCPU); err == nil {
						for _, v := range perCPU { agg.Packets += v.Packets; agg.Bytes += v.Bytes }
						pkts, bytes = agg.Packets, agg.Bytes
					}
				}
				deltaPkts := float64(0)
				deltaBytes := float64(0)
				if pkts >= lastPkts { deltaPkts = float64(pkts - lastPkts) }
				if bytes >= lastBytes { deltaBytes = float64(bytes - lastBytes) }
				lastPkts, lastBytes = pkts, bytes

				labels := metric.WithAttributes() // add attrs (device, port, direction) if desired
				if deltaPkts > 0 { pktCtr.Add(ctx, deltaPkts, labels) }
				if deltaBytes > 0 { byteCtr.Add(ctx, deltaBytes, labels) }
			}
		}
	}()

	return &results{}, nil
}

type results struct{}

func (r *results) Summary() interface{} {
	return map[string]any{"ok": true}
}

