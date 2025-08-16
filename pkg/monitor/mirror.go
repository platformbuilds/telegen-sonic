//go:build linux

package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Mirror creates (or reuses) a traffic mirror target interface that the
// collector will attach to. Preferred mode is ERSPAN v2 with a dedicated
// netdev (e.g. "erspan0"). If ERSPAN env config is not present, we fall
// back to a harmless placeholder that simply returns "erspan0".
//
// Environment (all optional unless noted):
//
//	TELEGEN_MIRROR_MODE        = "erspan" | "placeholder"   (default: erspan)
//	TELEGEN_ERSPAN_NAME        = name of netdev             (default: erspan0)
//	TELEGEN_ERSPAN_DEV         = source dev for mirroring   (default: spec.Port)
//	TELEGEN_ERSPAN_REMOTE      = remote IPv4 address        (REQUIRED for erspan)
//	TELEGEN_ERSPAN_LOCAL       = local  IPv4 address        (REQUIRED for erspan)
//	TELEGEN_ERSPAN_KEY         = numeric key (ERSPAN session id) (default: 10)
//	TELEGEN_ERSPAN_TTL         = TTL value                  (default: 64)
//	TELEGEN_ERSPAN_TOS         = TOS/DSCP (e.g. "inherit")  (default: inherit)
//
// Example (env):
//
//	TELEGEN_MIRROR_MODE=erspan
//	TELEGEN_ERSPAN_REMOTE=10.0.0.100
//	TELEGEN_ERSPAN_LOCAL=10.0.0.10
//	TELEGEN_ERSPAN_DEV=Ethernet0
//	TELEGEN_ERSPAN_KEY=17
//	TELEGEN_ERSPAN_TTL=64
//	TELEGEN_ERSPAN_NAME=erspan0
//
// NOTE: This function assumes the container has CAP_NET_ADMIN and `ip`.
//
//	On failure (or when not configured) it falls back to placeholder.
type Mirror struct{}

func (m *Mirror) Create(spec JobSpec) (string, func() error, error) {
	mode := getenvDefault("TELEGEN_MIRROR_MODE", "erspan")
	if strings.EqualFold(mode, "erspan") {
		ifname, cleanup, err := ensureERSPAN(spec)
		if err == nil {
			fmt.Printf("Created ERSPAN mirror for port=%s dir=%s -> %s\n", spec.Port, spec.Direction, ifname)
			return ifname, cleanup, nil
		}
		// If ERSPAN was requested but provisioning failed, surface the error but
		// still return a safe placeholder so CI/dev can proceed if desired.
		fmt.Printf("ERSPAN provisioning failed: %v\n", err)
	}

	// Placeholder: no real mirroring; return a stable name to allow tc attach attempts.
	ifname := getenvDefault("TELEGEN_ERSPAN_NAME", "erspan0")
	fmt.Printf("Created mirror session (placeholder) for port=%s dir=%s -> %s\n", spec.Port, spec.Direction, ifname)
	cleanup := func() error {
		fmt.Println("Deleted mirror session (placeholder)")
		return nil
	}
	return ifname, cleanup, nil
}

/* ------------------ ERSPAN helpers ------------------ */

func ensureERSPAN(spec JobSpec) (string, func() error, error) {
	name := getenvDefault("TELEGEN_ERSPAN_NAME", "erspan0")
	dev := getenvDefault("TELEGEN_ERSPAN_DEV", spec.Port)
	remote := os.Getenv("TELEGEN_ERSPAN_REMOTE")
	local := os.Getenv("TELEGEN_ERSPAN_LOCAL")
	key := getenvDefault("TELEGEN_ERSPAN_KEY", "10")
	ttl := getenvDefault("TELEGEN_ERSPAN_TTL", "64")
	tos := getenvDefault("TELEGEN_ERSPAN_TOS", "inherit") // "inherit" or numeric

	if remote == "" || local == "" {
		return "", nil, fmt.Errorf("missing TELEGEN_ERSPAN_REMOTE or TELEGEN_ERSPAN_LOCAL")
	}
	if dev == "" {
		return "", nil, fmt.Errorf("missing TELEGEN_ERSPAN_DEV (or JobSpec.Port)")
	}

	// If already exists, reuse.
	if linkExists(name) {
		return name, func() error { return nil }, nil
	}

	// ip link add erspan device:
	// erspan v2 with key, remote/local, dev, ttl, tos/dscp inherit
	args := []string{
		"link", "add", "name", name, "type", "erspan",
		"erspan_ver", "2",
		"key", key,
		"remote", remote,
		"local", local,
		"dev", dev,
		"ttl", ttl,
	}
	if tos != "" {
		// "inherit" is valid; numeric DSCP/TOS also supported
		args = append(args, "tos", tos)
	}

	if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("ip link add %s failed: %v: %s", name, err, string(out))
	}
	if out, err := exec.Command("ip", "link", "set", name, "up").CombinedOutput(); err != nil {
		_ = exec.Command("ip", "link", "del", name).Run()
		return "", nil, fmt.Errorf("ip link set up %s failed: %v: %s", name, err, string(out))
	}

	cleanup := func() error {
		_ = exec.Command("ip", "link", "del", name).Run()
		return nil
	}
	return name, cleanup, nil
}

func linkExists(name string) bool {
	cmd := exec.Command("ip", "link", "show", "dev", name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func getenvDefault(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
