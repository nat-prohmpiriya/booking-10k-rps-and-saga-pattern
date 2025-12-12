package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// KafkaTracerName is the name of the Kafka tracer
	KafkaTracerName = "kafka-client"
)

// KafkaHeadersCarrier implements propagation.TextMapCarrier for Kafka headers
type KafkaHeadersCarrier map[string]string

// Get returns the value for a given key
func (c KafkaHeadersCarrier) Get(key string) string {
	return c[key]
}

// Set sets a key-value pair
func (c KafkaHeadersCarrier) Set(key, value string) {
	c[key] = value
}

// Keys returns all keys in the carrier
func (c KafkaHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectKafkaHeaders injects trace context into Kafka message headers
func InjectKafkaHeaders(ctx context.Context, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, KafkaHeadersCarrier(headers))
	return headers
}

// ExtractKafkaContext extracts trace context from Kafka message headers
func ExtractKafkaContext(ctx context.Context, headers map[string]string) context.Context {
	if headers == nil {
		return ctx
	}
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, KafkaHeadersCarrier(headers))
}

// StartProducerSpan starts a span for Kafka producer
func StartProducerSpan(ctx context.Context, topic string, key string) (context.Context, trace.Span) {
	tracer := otel.Tracer(KafkaTracerName)
	spanName := topic + " publish"

	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKafka,
			semconv.MessagingDestinationName(topic),
			semconv.MessagingOperationTypePublish,
			attribute.String("messaging.kafka.message.key", key),
		),
	)

	return ctx, span
}

// StartConsumerSpan starts a span for Kafka consumer
func StartConsumerSpan(ctx context.Context, topic string, partition int32, offset int64, key string) (context.Context, trace.Span) {
	tracer := otel.Tracer(KafkaTracerName)
	spanName := topic + " process"

	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			semconv.MessagingSystemKafka,
			semconv.MessagingDestinationName(topic),
			semconv.MessagingOperationTypeProcess,
			attribute.Int("messaging.kafka.partition", int(partition)),
			attribute.Int64("messaging.kafka.message.offset", offset),
			attribute.String("messaging.kafka.message.key", key),
		),
	)

	return ctx, span
}

// EndKafkaSpanSuccess marks the span as successful and ends it
func EndKafkaSpanSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
	span.End()
}

// EndKafkaSpanError marks the span as failed with an error and ends it
func EndKafkaSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.End()
}

// AddKafkaProducerAttributes adds producer-specific attributes to span
func AddKafkaProducerAttributes(span trace.Span, partition int32, offset int64) {
	span.SetAttributes(
		attribute.Int("messaging.kafka.partition", int(partition)),
		attribute.Int64("messaging.kafka.message.offset", offset),
	)
}
