//go:build linux

package monitor

import "github.com/cilium/ebpf"

// ebpfMapShim aliases the real type so tests can pass a nil *ebpf.Map
// to NewMetricsCollector without creating kernel objects.
type ebpfMapShim = ebpf.Map
