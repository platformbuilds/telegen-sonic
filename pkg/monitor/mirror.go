package monitor

import (
	"fmt"
	"os/exec"
)

// Mirror sets up a CPU-visible interface that receives copies of data-plane packets.
// span_method=span uses SONiC CLI 'config mirror_session ... to_cpu'. 
// span_method=erspan creates a local ERSPAN tunnel (erspan0) that loops to itself.
type Mirror struct {
	MgmtIP string // required for erspan self-loop
}

func (m *Mirror) Create(spec JobSpec) (string, func() error, error) {
	if spec.SpanMethod == "erspan" {
		if m.MgmtIP == "" { m.MgmtIP = "127.0.0.1" }
		ifname := "erspan0"
		// Best-effort: create erspan0 if missing (id=100, key=100)
		cmd := exec.Command("bash", "-lc", fmt.Sprintf("ip link show %s || ip link add %s type erspan seq key 100 local %s remote %s", ifname, ifname, m.MgmtIP, m.MgmtIP))
		out, err := cmd.CombinedOutput()
		if err != nil { return "", nil, fmt.Errorf("erspan setup failed: %v: %s", err, string(out)) }
		_ = exec.Command("ip", "link", "set", ifname, "up").Run()
		cleanup := func() error {
			_ = exec.Command("ip", "link", "del", ifname).Run()
			return nil
		}
		return ifname, cleanup, nil
	}
	// Default: SPAN to CPU via SONiC CLI, using to_cpu session name
	// NOTE: This is a placeholder. In production, program a named session that mirrors the given port/direction/VLAN.
	// We simply return the Linux ifname that typically receives CPU traffic for mirrored frames, e.g., 'Ethernet0' mirror to 'lo' is platform-specific.
	ifname := "mirror0"
	fmt.Printf("[WARN] SPAN placeholder active; returning %s. Implement vendor-specific session setup.\n", ifname)
	cleanup := func() error { fmt.Println("mirror session cleanup (noop)"); return nil }
	return ifname, cleanup, nil
}
