package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"github.com/vihan/taskorchestrator/internal/infrastructure/config"
)

// InitializeTracing sets up OpenTelemetry tracing with Zipkin exporter
func InitializeTracing(ctx context.Context, cfg *config.ObservabilityConfig, appName string) (*trace.TracerProvider, error) {
	if !cfg.TracingEnabled {
		// Return a no-op tracer provider if tracing is disabled
		return trace.NewTracerProvider(), nil
	}

	exporter, err := zipkin.New(
		cfg.ZipkinEndpoint,
		zipkin.WithLogger(nil), // Suppress internal logging
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zipkin exporter: %w", err)
	}

	res := resource.NewWithAttributes(
		ctx,
		semconv.ServiceNameKey.String(appName),
	)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

// ShutdownTracing shuts down the tracer provider
func ShutdownTracing(ctx context.Context, tp *trace.TracerProvider) error {
	return tp.Shutdown(ctx)
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
