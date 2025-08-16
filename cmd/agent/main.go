//go:build linux

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/platformbuilds/telegen-sonic/pkg/api"
	"github.com/platformbuilds/telegen-sonic/pkg/monitor"
)

func main() {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	// 1) Set up OTel metrics
	ctx := context.Background()
	mp, meter, err := monitor.SetupOTelMetrics(
		ctx,
		"telegen-sonic", // service.name
		endpoint,
		false,          // insecure
		10*time.Second, // export interval
	)
	if err != nil {
		log.Fatalf("otel setup failed: %v", err)
	}
	defer func() {
		shctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mp.Shutdown(shctx)
	}()

	// 2) Open pinned BPF maps (ok if missing; your loader may pin them later)
	statsMap, ifStatsMap, err := monitor.OpenPinnedMaps(monitor.DefaultPinDir)
	if err != nil {
		log.Printf("warning: could not open pinned maps: %v", err)
	}

	// 3) Metrics collector + adapter (implements Supervisor's Collector interface)
	mc, err := monitor.NewMetricsCollector(meter, statsMap, ifStatsMap, 5*time.Second)
	if err != nil {
		log.Fatalf("collector init failed: %v", err)
	}
	col := monitor.NewBPFCollector(mc)

	// 4) Your providers (replace with your real implementations if different)
	mir := &monitor.Mirror{} // implements MirrorProvider
	att := &monitor.TC{}     // implements AttachProvider

	// 5) Supervisor (implements api.Core) — pass it straight to the API handlers
	sup := monitor.NewSupervisor(mir, att, col, 2) // Supervisor

	// Use the adapter (implements api.Core)
	core := &monitor.CoreAdapter{S: sup}

	h := &api.Handlers{Core: core} // api.Core satisfied
	r := api.NewRouter(h)

	log.Println("listening on 127.0.0.1:8080")
	if err := http.ListenAndServe("127.0.0.1:8080", r); err != nil {
		log.Fatal(err)
	}
}
