//go:build linux

package system

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRun_Success(t *testing.T) {
	// Prints to stdout and stderr, exits 0
	stdout, stderr, err := Run(context.Background(), "/bin/sh", "-c", "printf foo; printf bar 1>&2")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if strings.TrimSpace(stdout) != "foo" {
		t.Fatalf("stdout = %q, want %q", stdout, "foo")
	}
	if strings.TrimSpace(stderr) != "bar" {
		t.Fatalf("stderr = %q, want %q", stderr, "bar")
	}
}

func TestRun_NonZeroExit(t *testing.T) {
	// Prints to both streams, exits with status 13
	stdout, stderr, err := Run(context.Background(), "/bin/sh", "-c", "printf out; printf err 1>&2; exit 13")
	if err == nil {
		t.Fatalf("expected non-nil error for non-zero exit")
	}
	if strings.TrimSpace(stdout) != "out" {
		t.Fatalf("stdout = %q, want %q", stdout, "out")
	}
	if strings.TrimSpace(stderr) != "err" {
		t.Fatalf("stderr = %q, want %q", stderr, "err")
	}
}

func TestRun_CancelContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Start something that would block if not canceled
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		stdout, stderr, err := Run(ctx, "/bin/sh", "-c", "sleep 5; printf done; printf err 1>&2")
		// We expect a context cancellation error
		if err == nil {
			t.Errorf("expected error due to cancel, got nil (stdout=%q, stderr=%q)", stdout, stderr)
		}
	}()
	// Cancel shortly after starting
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-doneCh
}

func TestRunTimeout_DeadlineExceeded(t *testing.T) {
	// Sleep longer than the timeout to trigger deadline exceeded
	stdout, stderr, err := RunTimeout(100*time.Millisecond, "/bin/sh", "-c", "sleep 1; printf late; printf err 1>&2")
	if err == nil {
		t.Fatalf("expected deadline exceeded error, got nil (stdout=%q, stderr=%q)", stdout, stderr)
	}
	// Depending on timing, stdout/stderr may be empty (process killed before printing)
}
