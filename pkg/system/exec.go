package system

import (
	"bytes"
	"context"
	"os/exec"
	time "time"
)

// Run executes the command and returns collected stdout/stderr.
// It waits for the process to exit on all paths (including cancel/timeout)
// to avoid data races on the buffers.
func Run(ctx context.Context, name string, args ...string) (string, string, error) {
	c := exec.CommandContext(ctx, name, args...)

	var out, errb bytes.Buffer
	c.Stdout = &out
	c.Stderr = &errb

	// Start the process
	if err := c.Start(); err != nil {
		return out.String(), errb.String(), err
	}

	// Wait in a goroutine; deliver its result on 'done'
	done := make(chan error, 1)
	go func() {
		done <- c.Wait()
	}()

	select {
	case e := <-done:
		// Process finished; safe to read buffers
		return out.String(), errb.String(), e

	case <-ctx.Done():
		// Kill and ensure the process has fully exited before returning
		_ = c.Process.Kill()
		<-done // wait for Wait() to complete to avoid races on buffers
		return out.String(), errb.String(), ctx.Err()
	}
}

// RunTimeout runs a command with a deadline and returns collected output.
func RunTimeout(d time.Duration, name string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return Run(ctx, name, args...)
}
