//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

/*
We test the real `main()` in a child process so calls to os.Exit inside the CLI
don't kill the test runner. The parent process starts an HTTP server that binds
to 127.0.0.1:8080 (the CLI's hardcoded base) and then runs the child with
arguments after a `--` sentinel.

The child re-enters this test binary with -test.run=TestCLIMain_Helper and
env CLI_E2E=1; that helper calls main() with the supplied args.
*/

func runCLI(t *testing.T, args ...string) (exitCode int, stdout string, stderr string) {
	t.Helper()

	// Start a dedicated HTTP server on 127.0.0.1:8080 for the CLI to call.
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("listen 127.0.0.1:8080: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/monitor/jobs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"job_id":"j1","status":"started","interface":"eth0"}`))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/monitor/jobs/j1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"job_id": "j1", "status": "running",
				"port": "Ethernet0", "interface": "eth0",
				"started_at": time.Now().UTC().Format(time.RFC3339Nano),
				"expires_at": time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339Nano),
			})
		case http.MethodDelete:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"job_id":"j1","status":"stopped"}`))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/v1/monitor/jobs/j1/results", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"window_sec":60,"packets_total":100,"bytes_total":200}`))
	})
	srv := &http.Server{Handler: mux}
	defer srv.Close()
	go func() { _ = srv.Serve(ln) }()

	// Build child args: re-enter test binary and trigger helper
	childArgs := []string{"-test.run=TestCLIMain_Helper", "--"}
	childArgs = append(childArgs, args...)

	cmd := exec.Command(os.Args[0], childArgs...)
	cmd.Env = append(os.Environ(), "CLI_E2E=1")
	out, errOut := &strings.Builder{}, &strings.Builder{}
	cmd.Stdout, cmd.Stderr = out, errOut

	err = cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			// Unexpected error running the subprocess
			t.Fatalf("run child: %v", err)
		}
	}
	return code, out.String(), errOut.String()
}

// This helper actually invokes main() in the child process.
func TestCLIMain_Helper(t *testing.T) {
	if os.Getenv("CLI_E2E") != "1" {
		return
	}
	// Locate the sentinel and rebuild os.Args for the CLI.
	args := os.Args
	for i, a := range args {
		if a == "--" {
			os.Args = append([]string{args[0]}, args[i+1:]...)
			break
		}
	}
	main()
}

func TestCLI_StartStatusResultsStop_Success(t *testing.T) {
	// start
	code, out, _ := runCLI(t, "start", "Ethernet0", "5")
	if code != 0 {
		t.Fatalf("start exit code=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"job_id":"j1"`) {
		t.Fatalf("start output missing job_id: %s", out)
	}

	// status
	code, out, _ = runCLI(t, "status", "j1")
	if code != 0 || !strings.Contains(out, `"status":"running"`) {
		t.Fatalf("status failed: code=%d out=%s", code, out)
	}

	// results
	code, out, _ = runCLI(t, "results", "j1")
	if code != 0 || !strings.Contains(out, `"packets_total":100`) {
		t.Fatalf("results failed: code=%d out=%s", code, out)
	}

	// stop
	code, out, _ = runCLI(t, "stop", "j1")
	if code != 0 || !strings.Contains(out, `"status":"stopped"`) {
		t.Fatalf("stop failed: code=%d out=%s", code, out)
	}
}

func TestCLI_UsageAndBadArgs(t *testing.T) {
	// No args -> usage -> exit 1
	code, _, _ := runCLI(t /* no args */)
	if code != 1 {
		t.Fatalf("expected usage exit code=1, got %d", code)
	}

	// Bad subcommand -> usage -> exit 1
	code, _, _ = runCLI(t, "unknown")
	if code != 1 {
		t.Fatalf("expected exit 1 for bad subcommand, got %d", code)
	}

	// start missing args -> usage -> exit 1
	code, _, _ = runCLI(t, "start", "Ethernet0")
	if code != 1 {
		t.Fatalf("expected exit 1 for start missing args, got %d", code)
	}
}

func TestCLI_Atoi(t *testing.T) {
	if atoi("42") != 42 || atoi("notnum") != 0 {
		t.Fatalf("atoi unexpected result")
	}
}
