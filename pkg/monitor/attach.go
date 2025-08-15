package monitor

import (
	"fmt"
	"os"
	"os/exec"
)

type TC struct{}

func (t *TC) Attach(ifname string, spec JobSpec) (func() error, error) {
	// Ensure clsact
	_ = exec.Command("tc", "qdisc", "add", "dev", ifname, "clsact").Run()

	// Load and attach via tc (bpf object expected at /bpf/tc_ingress.bpf.o inside container)
	obj := "/bpf/tc_ingress.bpf.o"
	if _, err := os.Stat(obj); err != nil {
		return nil, fmt.Errorf("missing BPF object: %s", obj)
	}
	// Attach to ingress
	cmd := exec.Command("tc", "filter", "replace", "dev", ifname, "ingress",
		"prio", "1", "handle", "1", "bpf", "da", "obj", obj, "sec", "classifier")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tc attach failed: %v: %s", err, string(out))
	}
	cleanup := func() error {
		_ = exec.Command("tc", "filter", "del", "dev", ifname, "ingress").Run()
		_ = exec.Command("tc", "qdisc", "del", "dev", ifname, "clsact").Run()
		return nil
	}
	fmt.Printf("Attached tc program on %s (dir=%s)\n", ifname, spec.Direction)
	return cleanup, nil
}
