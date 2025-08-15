package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func usage() {
	fmt.Println("telegen-sonic start|status|results|stop ...")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 { usage() }
	switch os.Args[1] {
	case "start":
		// minimal: telegen-sonic start PORT DURATION_SEC
		if len(os.Args) < 4 { usage() }
		req := map[string]interface{}{
			"port": os.Args[2], "direction": "ingress", "span_method": "span",
			"duration_sec": atoi(os.Args[3]), "sample_rate": 100, "otlp_export": true, "result_detail": "summary",
		}
		call("POST", "/v1/monitor/jobs", req)
	case "status":
		if len(os.Args) < 3 { usage() }
		call("GET", "/v1/monitor/jobs/"+os.Args[2], nil)
	case "results":
		if len(os.Args) < 3 { usage() }
		call("GET", "/v1/monitor/jobs/"+os.Args[2]+"/results?format=json", nil)
	case "stop":
		if len(os.Args) < 3 { usage() }
		call("DELETE", "/v1/monitor/jobs/"+os.Args[2], nil)
	default:
		usage()
	}
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func call(method, path string, body interface{}) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	url := "http://127.0.0.1:8080" + path
	req, _ := http.NewRequest(method, url, rd)
	if body != nil { req.Header.Set("Content-Type", "application/json") }
	resp, err := http.DefaultClient.Do(req)
	if err != nil { fmt.Println("ERR:", err); os.Exit(2) }
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
	fmt.Println()
	if resp.StatusCode >= 400 { os.Exit(1) }
}
