package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/platformbuilds/sonic-dpmon/pkg/api"
	"github.com/platformbuilds/sonic-dpmon/pkg/monitor"
)

type CoreAdapter struct{ S *monitor.Supervisor }

func (c *CoreAdapter) TryStartJob(req api.StartJobRequest) (api.StartJobResponse, int, error) {
	resp, code, err := c.S.TryStartJob(struct{ api.StartJobRequest }{req})
	if err != nil { return api.StartJobResponse{}, code, err }
	m := resp.(map[string]interface{})
	return api.StartJobResponse{
		JobID: m["job_id"].(string), Status: m["status"].(string), Interface: m["interface"].(string),
	}, code, nil
}
func (c *CoreAdapter) GetJob(id string) (api.JobStatus, int, error) {
	resp, code, err := c.S.GetJob(id)
	if err != nil { return api.JobStatus{}, code, err }
	m := resp.(map[string]interface{})
	return api.JobStatus{
		JobID: m["job_id"].(string), Status: m["status"].(string),
		Port: m["port"].(string), Interface: m["interface"].(string),
	}, code, nil
}
func (c *CoreAdapter) StopJob(id string) (api.StopJobResponse, int, error) {
	resp, code, err := c.S.StopJob(id)
	if err != nil { return api.StopJobResponse{}, code, err }
	m := resp.(map[string]interface{})
	return api.StopJobResponse{ JobID: m["job_id"].(string), Status: m["status"].(string) }, code, nil
}
func (c *CoreAdapter) GetResults(id string) (api.JobResults, int, error) {
	_, code, err := c.S.GetResults(id)
	if err != nil { return api.JobResults{}, code, err }
	return api.JobResults{}, code, nil
}

func main() {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" { endpoint = "localhost:4317" }
	shutdown, err := monitor.SetupOTel(endpoint, 10*time.Second)
	if err != nil { log.Fatalf("otel setup failed: %v", err) }
	defer shutdown(nil)

	mir := &monitor.Mirror{}
	att := &monitor.TC{}
	col := &monitor.CollectorImpl{}
	sup := monitor.NewSupervisor(mir, att, col, 2) // hard limit 2
	adapter := &CoreAdapter{S: sup}
	h := &api.Handlers{Core: adapter}
	r := api.NewRouter(h)
	log.Println("listening on 127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", r))
}
