// Package observability provides OpenTelemetry tracing for A2A Gateway
package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer initializes the OpenTelemetry tracer
func InitTracer(serviceName, endpoint string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating exporter: %w", err)
	}

	// Create resource with service attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("0.1.0"),
			attribute.String("deployment.environment", "production"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	// Create tracer provider with batch processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(0.1), // 10% sampling
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

// Tracer returns the global tracer
func Tracer() trace.Tracer {
	return otel.Tracer("a2a-gateway")
}

// A2A attributes for tracing
func A2AAttributes(agentID, taskID string, promptTokens, completionTokens int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("agent.id", agentID),
		attribute.String("task.id", taskID),
		attribute.Int("prompt.tokens", promptTokens),
		attribute.Int("completion.tokens", completionTokens),
		attribute.Int("total.tokens", promptTokens+completionTokens),
	}
}

// TrustAttributes adds trust-related attributes to spans
func TrustAttributes(trustScore float64, jurisdiction string, violations int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Float64("trust.score", trustScore),
		attribute.String("jurisdiction", jurisdiction),
		attribute.Int("policy.violations", violations),
	}
}

// ProtocolAttributes adds A2A protocol attributes
func ProtocolAttributes(protocolVersion, capability string, streaming bool) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("a2a.protocol.version", protocolVersion),
		attribute.String("a2a.capability", capability),
		attribute.Bool("a2a.streaming", streaming),
	}
}

// StartTaskSpan starts a new span for an A2A task
func StartTaskSpan(ctx context.Context, taskID, agentID, operation string) (context.Context, trace.Span) {
	attrs := A2AAttributes(agentID, taskID, 0, 0)
	return Tracer().Start(ctx, operation,
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindServer),
	)
}

// RecordTokenUsage records token usage on a span
func RecordTokenUsage(span trace.Span, promptTokens, completionTokens int) {
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int("prompt.tokens", promptTokens),
			attribute.Int("completion.tokens", completionTokens),
			attribute.Int("total.tokens", promptTokens+completionTokens),
		)
	}
}
