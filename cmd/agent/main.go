package main

import (
	"log"
	"net/http"
	"os"
	"github.com/platformbuilds/sonic-dpmon/pkg/api"
	"github.com/platformbuilds/sonic-dpmon/pkg/monitor"
)

type CoreAdapter struct{ S *monitor.Supervisor }

func (c *CoreAdapter) TryStartJob(req api.StartJobRequest) (api.StartJobResponse, int, error) {
	resp, code, err := c.S.TryStartJob(struct{ api.StartJobRequest }{req})
	if err != nil { return api.StartJobResponse{}, code, err }
	m := resp.(map[string]interface{})
	return api.StartJobResponse{ JobID: m["job_id"].(string), Status: m["status"].(string), Interface: m["interface"].(string) }, code, nil
}
func (c *CoreAdapter) GetJob(id string) (api.JobStatus, int, error) {
	resp, code, err := c.S.GetJob(id)
	if err != nil { return api.JobStatus{}, code, err }
	m := resp.(map[string]interface{})
	return api.JobStatus{ JobID: m["job_id"].(string), Status: m["status"].(string), Port: m["port"].(string), Interface: m["interface"].(string) }, code, nil
}
func (c *CoreAdapter) StopJob(id string) (api.StopJobResponse, int, error) {
	resp, code, err := c.S.StopJob(id)
	if err != nil { return api.StopJobResponse{}, code, err }
	m := resp.(map[string]interface{})
	return api.StopJobResponse{ JobID: m["job_id"].(string), Status: m["status"].(string) }, code, nil
}
func (c *CoreAdapter) GetResults(id string) (api.JobResults, int, error) {
	resp, code, err := c.S.GetResults(id)
	if err != nil { return api.JobResults{}, code, err }
	m := resp.(map[string]interface{})
	lat := m["latency_histogram_ns"].(map[string]interface{})
	b := []uint64{}; cnt := []uint64{}
	return api.JobResults{
		WindowSec: int(m["window_sec"].(int)),
		Packets: 0, Bytes: 0, Errors: map[string]uint64{},
		LatencyHistogramNs: api.Histogram{Bounds: b, Counts: cnt},
		OTLPExport: api.OTLPInfo{Exported: true, Endpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")},
	}, code, nil
}

func main() {
	mgmt := os.Getenv("MGMT_IP")
	mir := &monitor.Mirror{MgmtIP: mgmt}
	att := &monitor.TC{}
	col := monitor.NewCollector()
	sup := monitor.NewSupervisor(mir, att, col, 2) // hard limit 2
	adapter := &CoreAdapter{S: sup}
	h := &api.Handlers{Core: adapter}
	r := api.NewRouter(h)
	addr := "127.0.0.1:8080"
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
