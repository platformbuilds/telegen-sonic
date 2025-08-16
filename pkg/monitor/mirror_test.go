//go:build linux

package monitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withFakeIP(t *testing.T, script string) (restore func(), logPath string) {
	t.Helper()
	td := t.TempDir()
	ip := filepath.Join(td, "ip")
	logPath = filepath.Join(td, "ip_calls.log")

	// Wrap given script to also log argv to a file
	wrapper := "#!/usr/bin/env bash\n" +
		"echo \"$0 $@\" >> " + logPath + "\n" +
		script + "\n"

	if err := os.WriteFile(ip, []byte(wrapper), 0o755); err != nil {
		t.Fatalf("write fake ip: %v", err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", td+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	return func() { _ = os.Setenv("PATH", oldPath) }, logPath
}

func TestMirror_ERSPAN_Success(t *testing.T) {
	/*
		Fake "ip" behavior:
		  - "ip link show dev <name>" → exit 1 (not existing yet)
		  - "ip link add name <name> type erspan ..." → exit 0
		  - "ip link set <name> up" → exit 0
	*/
	script := `
# return 1 for "ip link show dev <name>" to simulate non-existent device
if [ "$1" = "link" ] && [ "$2" = "show" ] && [ "$3" = "dev" ]; then
  exit 1
fi
# allow "ip link add ..." and "ip link set ... up"
if [ "$1" = "link" ] && [ "$2" = "add" ]; then exit 0; fi
if [ "$1" = "link" ] && [ "$2" = "set" ] && [ "$4" = "up" ]; then exit 0; fi
# default: success
exit 0
`
	restore, logPath := withFakeIP(t, script)
	defer restore()

	// Set ERSPAN env
	t.Setenv("TELEGEN_MIRROR_MODE", "erspan")
	t.Setenv("TELEGEN_ERSPAN_NAME", "erspan0")
	t.Setenv("TELEGEN_ERSPAN_REMOTE", "192.0.2.100")
	t.Setenv("TELEGEN_ERSPAN_LOCAL", "192.0.2.10")
	t.Setenv("TELEGEN_ERSPAN_DEV", "Ethernet0")
	t.Setenv("TELEGEN_ERSPAN_KEY", "42")
	t.Setenv("TELEGEN_ERSPAN_TTL", "64")

	m := &Mirror{}
	ifname, cleanup, err := m.Create(JobSpec{Port: "Ethernet0", Direction: "ingress"})
	if err != nil {
		t.Fatalf("Mirror.Create error: %v", err)
	}
	if ifname != "erspan0" {
		t.Fatalf("expected erspan0, got %q", ifname)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup")
	}
	_ = cleanup()

	// Assert that our fake ip was called to add and set the erspan link
	data, _ := os.ReadFile(logPath)
	log := string(data)
	if !strings.Contains(log, "ip link add name erspan0 type erspan") {
		t.Fatalf("expected 'ip link add name erspan0 type erspan' in calls; got:\n%s", log)
	}
	if !strings.Contains(log, "ip link set erspan0 up") {
		t.Fatalf("expected 'ip link set erspan0 up' in calls; got:\n%s", log)
	}
}

func TestMirror_Placeholder_Mode(t *testing.T) {
	// No fake ip needed; placeholder does not call ip(8)
	t.Setenv("TELEGEN_MIRROR_MODE", "placeholder")
	t.Setenv("TELEGEN_ERSPAN_NAME", "erspan0")

	m := &Mirror{}
	ifname, cleanup, err := m.Create(JobSpec{Port: "Ethernet0", Direction: "ingress"})
	if err != nil {
		t.Fatalf("Mirror.Create error: %v", err)
	}
	if ifname != "erspan0" {
		t.Fatalf("expected erspan0 in placeholder mode, got %q", ifname)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup in placeholder mode")
	}
	_ = cleanup()
}

func TestMirror_ERSPAN_MissingEnv_FallsBackToPlaceholder(t *testing.T) {
	// Ask for ERSPAN but omit REMOTE/LOCAL -> should gracefully fall back
	t.Setenv("TELEGEN_MIRROR_MODE", "erspan")
	t.Setenv("TELEGEN_ERSPAN_NAME", "erspanX") // verify name is propagated to placeholder

	m := &Mirror{}
	ifname, cleanup, err := m.Create(JobSpec{Port: "Ethernet0", Direction: "ingress"})
	if err != nil {
		t.Fatalf("Mirror.Create returned error; expected fallback, got: %v", err)
	}
	if ifname != "erspanX" {
		t.Fatalf("expected fallback to placeholder with name erspanX, got %q", ifname)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup in fallback")
	}
	_ = cleanup()
}
