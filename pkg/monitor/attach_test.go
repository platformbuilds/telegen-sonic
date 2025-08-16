//go:build linux

package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func withFakeTC(t *testing.T) (restore func()) {
	t.Helper()
	td := t.TempDir()
	tc := filepath.Join(td, "tc")
	// simple stub that always succeeds
	script := "#!/usr/bin/env bash\nexit 0\n"
	if err := os.WriteFile(tc, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tc: %v", err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", td+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	return func() { _ = os.Setenv("PATH", oldPath) }
}

func TestTC_Attach_SuccessWithEnvOverride(t *testing.T) {
	restore := withFakeTC(t)
	defer restore()

	// create a temp "bpf object"
	td := t.TempDir()
	obj := filepath.Join(td, "tc_ingress.bpf.o")
	if err := os.WriteFile(obj, []byte{0x7f, 'E', 'L', 'F'}, 0o644); err != nil {
		t.Fatalf("write temp obj: %v", err)
	}
	t.Setenv("TELEGEN_BPF_OBJ", obj)

	tc := &TC{}
	cleanup, err := tc.Attach("eth0", JobSpec{Direction: "ingress"})
	if err != nil {
		t.Fatalf("Attach returned error: %v", err)
	}
	if cleanup == nil {
		t.Fatalf("expected non-nil cleanup")
	}
	// cleanup should also succeed with stubbed tc
	if err := cleanup(); err != nil {
		t.Fatalf("cleanup error: %v", err)
	}
}

func TestTC_Attach_MissingObject(t *testing.T) {
	restore := withFakeTC(t)
	defer restore()

	td := t.TempDir()
	nonexistent := filepath.Join(td, "does-not-exist.o")
	t.Setenv("TELEGEN_BPF_OBJ", nonexistent)

	tc := &TC{}
	if _, err := tc.Attach("eth0", JobSpec{Direction: "ingress"}); err == nil {
		t.Fatalf("expected error for missing object, got nil")
	}
}
