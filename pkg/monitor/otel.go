//go:build linux

package monitor

import (
	"context"
	"crypto/tls"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
)

// SetupOTelMetrics configures an OTLP gRPC exporter + periodic reader and returns a MeterProvider and Meter.
func SetupOTelMetrics(ctx context.Context, serviceName, endpoint string, insecure bool, interval time.Duration) (*sdkmetric.MeterProvider, metric.Meter, error) {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
	}

	if insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	} else {
		creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(creds))
	}

	exp, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	// No semconv import; set attributes directly.
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(interval))

	// Example view: explicit buckets for bpf.bytes
	bytesHistView := sdkmetric.NewView(
		sdkmetric.Instrument{
			Name: "bpf.bytes",
			Kind: sdkmetric.InstrumentKindHistogram,
		},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{64, 128, 256, 512, 1024, 1500, 9000, 65536, 262144, 1048576},
			},
		},
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
		sdkmetric.WithView(bytesHistView),
	)
	otel.SetMeterProvider(mp)

	meter := mp.Meter("telegen-sonic/monitor")
	return mp, meter, nil
}
