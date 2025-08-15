package system

import (
	"bytes"
	"context"
	"os/exec"
	time "time"
)

func Run(ctx context.Context, name string, args ...string) (string, string, error) {
	c := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	c.Stdout = &out
	c.Stderr = &errb
	c.Start()
	done := make(chan error, 1)
	go func(){ done <- c.Wait() }()
	select {
	case e := <-done:
		return out.String(), errb.String(), e
	case <-ctx.Done():
		_ = c.Process.Kill()
		return out.String(), errb.String(), ctx.Err()
	}
}

func RunTimeout(d time.Duration, name string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return Run(ctx, name, args...)
}
