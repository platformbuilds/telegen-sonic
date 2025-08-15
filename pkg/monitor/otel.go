package monitor

import (
	"context"
	"log"
	"time"
	gootel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
)

type OTelExporter struct {
	mp *metric.MeterProvider
}

func NewOTelExporter(endpoint string) (*OTelExporter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(endpoint), otlpmetricgrpc.WithInsecure())
	if err != nil { return nil, err }
	mp := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exp)))
	gootel.SetMeterProvider(mp)
	return &OTelExporter{mp: mp}, nil
}

func (o *OTelExporter) Shutdown(ctx context.Context) error {
	if o.mp != nil { return o.mp.Shutdown(ctx) }
	return nil
}

func (c *CollectorImpl) WireOTel(endpoint string) {
	// TODO: register instruments and export counters/histograms from aggState
	log.Println("OTel exporter endpoint set to", endpoint)
}
