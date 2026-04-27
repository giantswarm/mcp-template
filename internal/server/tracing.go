package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Shutdown drains pending spans. Safe to call multiple times.
type Shutdown func(ctx context.Context) error

// InitTracing installs a global OTEL tracer provider when the standard
// OTEL_EXPORTER_OTLP_(TRACES_)?ENDPOINT env vars are set. Without an
// endpoint the propagator is still installed (incoming traceparents are
// honoured) and Shutdown is a no-op.
func InitTracing(ctx context.Context, serviceName, serviceVersion string) (Shutdown, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	endpoint := cmp.Or(
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"),
		os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	)
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := buildExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}
	res, err := buildResource(ctx, serviceName, serviceVersion)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

func buildExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	proto := strings.ToLower(cmp.Or(
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"),
		os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"),
		"http/protobuf",
	))
	switch proto {
	case "grpc":
		return otlptracegrpc.New(ctx)
	case "http/protobuf", "http":
		return otlptrace.New(ctx, otlptracehttp.NewClient())
	default:
		return nil, fmt.Errorf("unknown OTEL_EXPORTER_OTLP_PROTOCOL=%q (want grpc|http/protobuf)", proto)
	}
}

func buildResource(ctx context.Context, serviceName, serviceVersion string) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
	}
	if v := os.Getenv("POD_NAME"); v != "" {
		attrs = append(attrs, semconv.K8SPodName(v))
	}
	if v := os.Getenv("POD_NAMESPACE"); v != "" {
		attrs = append(attrs, semconv.K8SNamespaceName(v))
	}
	if v := os.Getenv("NODE_NAME"); v != "" {
		attrs = append(attrs, semconv.K8SNodeName(v))
	}
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		if errors.Is(err, resource.ErrPartialResource) && res != nil {
			slog.Default().Warn("otel resource: partial detection — continuing", "error", err)
			return res, nil
		}
		return nil, err
	}
	return res, nil
}
