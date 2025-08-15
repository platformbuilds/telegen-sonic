package monitor

import (
	"fmt"
)

type Mirror struct{}

func (m *Mirror) Create(spec JobSpec) (string, func() error, error) {
	// Placeholder: return a CPU-visible interface that will receive mirrored traffic.
	// For SONiC production, implement real SPAN/ERSPAN provisioning here and return the ifname.
	ifname := "erspan0" // or 'mirror0' for local SPAN target
	fmt.Printf("Created mirror session for port=%s dir=%s -> %s (placeholder)\n", spec.Port, spec.Direction, ifname)
	cleanup := func() error {
		fmt.Println("Deleted mirror session (placeholder)")
		return nil
	}
	return ifname, cleanup, nil
}
