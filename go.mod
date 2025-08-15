module github.com/platformbuilds/sonic-dpmon

go 1.23

require (
    github.com/cilium/ebpf v0.16.0
    github.com/go-chi/chi/v5 v5.0.10
    go.opentelemetry.io/otel v1.27.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.27.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.27.0
    go.opentelemetry.io/otel/sdk/metric v1.27.0
    github.com/google/uuid v1.6.0
    golang.org/x/sys v0.21.0
)
