package telemetry

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracerName is the name of the Gin tracer
	TracerName = "gin-server"

	// TraceIDHeader is the header key for trace ID
	TraceIDHeader = "X-Trace-ID"

	// SpanIDHeader is the header key for span ID
	SpanIDHeader = "X-Span-ID"
)

// TracingMiddleware returns a Gin middleware for automatic tracing using official otelgin
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName,
		otelgin.WithTracerProvider(otel.GetTracerProvider()),
		otelgin.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// TraceHeaderMiddleware adds trace ID and span ID to response headers
// Should be used after TracingMiddleware
func TraceHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Add trace ID to response header after request processing
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().HasTraceID() {
			traceID := span.SpanContext().TraceID().String()
			c.Header(TraceIDHeader, traceID)
			c.Set("trace_id", traceID)
		}
		if span.SpanContext().HasSpanID() {
			spanID := span.SpanContext().SpanID().String()
			c.Header(SpanIDHeader, spanID)
			c.Set("span_id", spanID)
		}
	}
}

// InjectTraceContext injects trace context into outgoing HTTP headers (for Gin context)
func InjectTraceContext(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(c.Request.Context(), propagation.MapCarrier(headers))
	return headers
}

// InjectTraceContextFromCtx injects trace context into map from standard context
func InjectTraceContextFromCtx(ctx context.Context) map[string]string {
	headers := make(map[string]string)
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.MapCarrier(headers))
	return headers
}

// ExtractTraceContext extracts trace context from headers into context
func ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, propagation.MapCarrier(headers))
}
