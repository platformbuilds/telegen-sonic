package monitor

import (
	"fmt"
	"os/exec"
)

// TC attaches eBPF object to tc clsact using the iproute2 `tc` command.
// Requires: clsact qdisc and tc with bpf support in the container/host.
type TC struct{}

func runTc(args ...string) error {
	cmd := exec.Command("tc", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tc %v failed: %v: %s", args, err, string(out))
	}
	return nil
}

func (t *TC) Attach(ifname string, spec JobSpec) (func() error, error) {
	// ensure clsact
	_ = runTc("qdisc", "add", "dev", ifname, "clsact")
	// Attach to ingress. If ResultDetail wants egress/both, add another filter.
	obj := "/bpf/tc_ingress.bpf.o" // expect object baked or volume mounted at /bpf
	if err := runTc("filter", "replace", "dev", ifname, "ingress", "prio", "1", "handle", "1", "bpf", "da", "obj", obj, "sec", "classifier"); err != nil {
		return nil, err
	}
	cleanup := func() error {
		_ = runTc("filter", "del", "dev", ifname, "ingress")
		_ = runTc("qdisc", "del", "dev", ifname, "clsact")
		return nil
	}
	fmt.Printf("Attached tc+BPF on %s\n", ifname)
	return cleanup, nil
}
