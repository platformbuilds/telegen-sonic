//go:build linux

package monitor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	DefaultPinDir = "/sys/fs/bpf/telegen-sonic"

	// Keep these in sync with bpf/tc_ingress.bpf.c
	idxIPv4  = 0
	idxIPv6  = 1
	idxICMP6 = 2
	idxOther = 3
	idxMax   = 4
)

type ProtoStats struct {
	Packets uint64
	Bytes   uint64
}

type IfProtoKey struct {
	Ifindex uint32
	Proto   uint32
}

// MetricsCollector periodically reads BPF maps and emits OTel metrics.
type MetricsCollector struct {
	statsMap   *ebpf.Map // BPF_MAP_TYPE_PERCPU_ARRAY [idxMax]ProtoStats
	ifStatsMap *ebpf.Map // BPF_MAP_TYPE_PERCPU_HASH {IfProtoKey: []ProtoStats per CPU}

	meter      otelmetric.Meter
	packetsCtr otelmetric.Int64Counter
	bytesHist  otelmetric.Int64Histogram

	lastGlobal [idxMax]ProtoStats
	lastIF     map[IfProtoKey]ProtoStats

	interval time.Duration
}

// OpenPinnedMaps expects pinned names "stats_percpu" and "if_stats_percpu".
func OpenPinnedMaps(pinDir string) (*ebpf.Map, *ebpf.Map, error) {
	if pinDir == "" {
		pinDir = DefaultPinDir
	}
	statsPath := filepath.Join(pinDir, "stats_percpu")
	ifStatsPath := filepath.Join(pinDir, "if_stats_percpu")

	statsMap, err := ebpf.LoadPinnedMap(statsPath, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("open stats_percpu: %w", err)
	}
	ifMap, err := ebpf.LoadPinnedMap(ifStatsPath, nil)
	if err != nil {
		// Per-interface map may be absent; that's fine.
		return statsMap, nil, nil
	}
	return statsMap, ifMap, nil
}

func NewMetricsCollector(meter otelmetric.Meter, statsMap, ifStatsMap *ebpf.Map, interval time.Duration) (*MetricsCollector, error) {
	if meter == nil {
		return nil, errors.New("meter is nil")
	}
	if statsMap == nil {
		return nil, errors.New("statsMap is nil")
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}

	packetsCtr, err := meter.Int64Counter(
		"bpf.packets",
		otelmetric.WithDescription("Packets observed by tc ingress eBPF"),
		otelmetric.WithUnit("1"), // dimensionless count
	)
	if err != nil {
		return nil, fmt.Errorf("create packets counter: %w", err)
	}

	bytesHist, err := meter.Int64Histogram(
		"bpf.bytes",
		otelmetric.WithDescription("Bytes observed by tc ingress eBPF"),
		otelmetric.WithUnit("By"), // bytes
	)
	if err != nil {
		return nil, fmt.Errorf("create bytes histogram: %w", err)
	}

	return &MetricsCollector{
		statsMap:   statsMap,
		ifStatsMap: ifStatsMap,
		meter:      meter,
		packetsCtr: packetsCtr,
		bytesHist:  bytesHist,
		lastIF:     make(map[IfProtoKey]ProtoStats),
		interval:   interval,
	}, nil
}

func (c *MetricsCollector) Start(ctx context.Context) error {
	t := time.NewTicker(c.interval)
	defer t.Stop()

	// First scrape establishes baselines.
	if err := c.collectOnce(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			_ = c.collectOnce(ctx) // keep going even if one scrape fails
		}
	}
}

func (c *MetricsCollector) collectOnce(ctx context.Context) error {
	// Global per-CPU ARRAY
	for idx := uint32(0); idx < idxMax; idx++ {
		sum, err := lookupPerCPUArray[ProtoStats](c.statsMap, idx)
		if err != nil {
			return fmt.Errorf("lookup stats_percpu[%d]: %w", idx, err)
		}
		dPackets := int64(diffU64(sum.Packets, c.lastGlobal[idx].Packets))
		dBytes := int64(diffU64(sum.Bytes, c.lastGlobal[idx].Bytes))
		c.lastGlobal[idx] = sum

		if dPackets > 0 || dBytes > 0 {
			attrs := []attribute.KeyValue{attribute.String("proto", protoName(idx))}
			if dPackets > 0 {
				c.packetsCtr.Add(ctx, dPackets, otelmetric.WithAttributes(attrs...))
			}
			if dBytes > 0 {
				c.bytesHist.Record(ctx, dBytes, otelmetric.WithAttributes(attrs...))
			}
		}
	}

	// Per-IF PERCPU_HASH (optional)
	if c.ifStatsMap != nil {
		numCPU := runtime.NumCPU()
		it := c.ifStatsMap.Iterate()
		var k IfProtoKey
		val := make([]ProtoStats, numCPU)

		for it.Next(&k, &val) {
			var agg ProtoStats
			for i := 0; i < len(val); i++ {
				agg.Packets += val[i].Packets
				agg.Bytes += val[i].Bytes
			}
			prev := c.lastIF[k]
			dPackets := int64(diffU64(agg.Packets, prev.Packets))
			dBytes := int64(diffU64(agg.Bytes, prev.Bytes))
			c.lastIF[k] = agg

			if dPackets > 0 || dBytes > 0 {
				attrs := []attribute.KeyValue{
					attribute.String("proto", protoName(uint32(k.Proto))),
					attribute.Int("ifindex", int(k.Ifindex)),
				}
				if dPackets > 0 {
					c.packetsCtr.Add(ctx, dPackets, otelmetric.WithAttributes(attrs...))
				}
				if dBytes > 0 {
					c.bytesHist.Record(ctx, dBytes, otelmetric.WithAttributes(attrs...))
				}
			}
		}
		if err := it.Err(); err != nil {
			return fmt.Errorf("iterate if_stats_percpu: %w", err)
		}
	}
	return nil
}

func protoName(idx uint32) string {
	switch idx {
	case idxIPv4:
		return "ipv4"
	case idxIPv6:
		return "ipv6"
	case idxICMP6:
		return "icmp6"
	default:
		return "other"
	}
}

// lookupPerCPUArray sums a PERCPU array element (key -> []T per CPU).
func lookupPerCPUArray[T any](m *ebpf.Map, key uint32) (T, error) {
	var zero T
	numCPU := runtime.NumCPU()
	vals := make([]T, numCPU)
	if err := m.Lookup(&key, &vals); err != nil {
		return zero, err
	}
	return sumSlice(vals), nil
}

func sumSlice[T any](in []T) (out T) {
	switch any(out).(type) {
	case ProtoStats:
		var s ProtoStats
		for _, v := range in {
			ps := any(v).(ProtoStats)
			s.Packets += ps.Packets
			s.Bytes += ps.Bytes
		}
		return any(s).(T)
	default:
		return out
	}
}

func diffU64(cur, prev uint64) uint64 {
	if cur >= prev {
		return cur - prev
	}
	return cur // wrapped (basic handling)
}
