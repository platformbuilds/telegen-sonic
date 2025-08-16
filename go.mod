module github.com/platformbuilds/telegen-sonic

go 1.23

require (
	github.com/cilium/ebpf v0.16.0
	github.com/go-chi/chi/v5 v5.0.10
	github.com/google/uuid v1.6.0
	go.opentelemetry.io/otel v1.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.27.0
	go.opentelemetry.io/otel/sdk/metric v1.27.0
	golang.org/x/sys v0.21.0
)

require golang.org/x/exp v0.0.0-20230224173230-c95f2b4c22f2 // indirect
