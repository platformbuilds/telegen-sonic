package monitor

import (
	"context"
	"log"
	"time"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/reader"
)

func SetupOTel(endpoint string, interval time.Duration) (func(context.Context) error, error) {
	exp, err := otlpmetricgrpc.New(context.Background(), otlpmetricgrpc.WithEndpoint(endpoint), otlpmetricgrpc.WithInsecure())
	if err != nil { return nil, err }
	prd := reader.NewPeriodicReader(exp, reader.WithInterval(interval))
	provider := metric.NewMeterProvider(
		metric.WithReader(prd),
		metric.WithView(metric.NewView(
			metric.Instrument{Name: "*"},
			metric.Stream{Aggregation: aggregation.Sum{}}, // defaults
		)),
	)
	otel.SetMeterProvider(provider)
	return provider.Shutdown, nil
}

