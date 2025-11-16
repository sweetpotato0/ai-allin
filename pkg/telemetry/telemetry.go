package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sweetpotato0/ai-allin/pkg/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config controls initialization of OpenTelemetry exporters.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Disable        bool
	Logger         *slog.Logger
}

// Init configures OpenTelemetry tracing based on the provided configuration.
// The returned shutdown function flushes exporters when the process exits.
func Init(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	if cfg.Disable {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "ai-allin"
	}
	logger := cfg.Logger
	if logger == nil {
		logger = logging.WithComponent("telemetry")
	}

	exp, err := newExporter(ctx, logger)
	if err != nil {
		return nil, err
	}

	resAttrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.ServiceName),
	}
	if cfg.ServiceVersion != "" {
		resAttrs = append(resAttrs, semconv.ServiceVersionKey.String(cfg.ServiceVersion))
	}
	if cfg.Environment != "" {
		resAttrs = append(resAttrs, attribute.String("environment", cfg.Environment))
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(resAttrs...),
		resource.WithFromEnv(),
		resource.WithOS(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("telemetry shutdown failed", "error", err)
			return err
		}
		return nil
	}, nil
}

func newExporter(ctx context.Context, logger *slog.Logger) (sdktrace.SpanExporter, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		logger.Warn("OTEL_EXPORTER_OTLP_ENDPOINT not set, using stdout trace exporter")
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create OTLP exporter: %w", err)
	}
	logger.Info("OTLP trace exporter configured", "endpoint", endpoint)
	return exp, nil
}

// End finalizes a span and captures the provided error.
func End(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, codes.Ok.String())
	}
	span.End()
}
