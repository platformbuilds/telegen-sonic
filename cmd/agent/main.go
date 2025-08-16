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

var (
	version = "dev"
	commit  = "unknown"
	date    = ""
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

	// 3) Metrics collector (runs globally in this process)
	mc, err := monitor.NewMetricsCollector(meter, statsMap, ifStatsMap, 5*time.Second)
	if err != nil {
		log.Fatalf("collector init failed: %v", err)
	}
	// Start the collector in the background so this single binary does API + metrics
	go func() {
		if err := mc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Printf("metrics collector stopped with error: %v", err)
		}
	}()

	// If Supervisor needs a Collector impl, wrap the already-running metrics collector.
	col := monitor.NewBPFCollector(mc)

	// 4) Your providers (replace with real implementations if different)
	mir := &monitor.Mirror{} // implements MirrorProvider
	att := &monitor.TC{}     // implements AttachProvider

	// 5) Supervisor and API wiring
	sup := monitor.NewSupervisor(mir, att, col, 2)
	core := &monitor.CoreAdapter{S: sup}

	h := &api.Handlers{Core: core}
	r := api.NewRouter(h)

	log.Println("listening on 127.0.0.1:8080")
	if err := http.ListenAndServe("127.0.0.1:8080", r); err != nil {
		log.Fatal(err)
	}
}
