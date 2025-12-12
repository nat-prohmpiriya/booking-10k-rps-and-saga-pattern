package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer represents a Kafka producer
type Producer struct {
	client *kgo.Client
	mu     sync.RWMutex
	closed bool
}

// ProducerConfig contains configuration for the Kafka producer
type ProducerConfig struct {
	Brokers       []string
	ClientID      string
	MaxRetries    int
	RetryInterval time.Duration
	BatchSize     int
	LingerMs      int
}

// Message represents a Kafka message
type Message struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Timestamp time.Time
}

// NewProducer creates a new Kafka producer
func NewProducer(ctx context.Context, cfg *ProducerConfig) (*Producer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("producer config is required")
	}

	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ProducerBatchMaxBytes(1024 * 1024), // 1MB
		kgo.RecordRetries(cfg.MaxRetries),
	}

	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}

	if cfg.BatchSize > 0 {
		opts = append(opts, kgo.MaxBufferedRecords(cfg.BatchSize))
	}

	if cfg.LingerMs > 0 {
		opts = append(opts, kgo.ProducerLinger(time.Duration(cfg.LingerMs)*time.Millisecond))
	}

	var client *kgo.Client
	var err error

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	retryInterval := cfg.RetryInterval
	if retryInterval <= 0 {
		retryInterval = 2 * time.Second
	}

	for i := 0; i < maxRetries; i++ {
		client, err = kgo.NewClient(opts...)
		if err == nil {
			// Ping to verify connection
			if pingErr := client.Ping(ctx); pingErr == nil {
				break
			} else {
				client.Close()
				err = pingErr
			}
		}

		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryInterval):
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer after %d retries: %w", maxRetries, err)
	}

	return &Producer{
		client: client,
	}, nil
}

// Produce sends a message to Kafka with optional tracing
func (p *Producer) Produce(ctx context.Context, msg *Message) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	// Start producer span
	ctx, span := telemetry.StartProducerSpan(ctx, msg.Topic, string(msg.Key))
	defer span.End()

	// Inject trace context into message headers
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	msg.Headers = telemetry.InjectKafkaHeaders(ctx, msg.Headers)

	record := &kgo.Record{
		Topic: msg.Topic,
		Key:   msg.Key,
		Value: msg.Value,
	}

	if !msg.Timestamp.IsZero() {
		record.Timestamp = msg.Timestamp
	}

	for key, value := range msg.Headers {
		record.Headers = append(record.Headers, kgo.RecordHeader{
			Key:   key,
			Value: []byte(value),
		})
	}

	result := p.client.ProduceSync(ctx, record)
	if err := result.FirstErr(); err != nil {
		telemetry.SetSpanError(ctx, err)
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Add partition and offset info to span
	if len(result) > 0 {
		r := result[0].Record
		telemetry.AddKafkaProducerAttributes(span, r.Partition, r.Offset)
	}

	return nil
}

// ProduceJSON serializes data to JSON and sends it to Kafka
func (p *Producer) ProduceJSON(ctx context.Context, topic string, key string, data interface{}, headers map[string]string) error {
	value, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	msg := &Message{
		Topic:     topic,
		Key:       []byte(key),
		Value:     value,
		Headers:   headers,
		Timestamp: time.Now(),
	}

	return p.Produce(ctx, msg)
}

// ProduceAsync sends a message asynchronously
func (p *Producer) ProduceAsync(ctx context.Context, msg *Message, callback func(error)) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		if callback != nil {
			callback(fmt.Errorf("producer is closed"))
		}
		return
	}
	p.mu.RUnlock()

	record := &kgo.Record{
		Topic: msg.Topic,
		Key:   msg.Key,
		Value: msg.Value,
	}

	if !msg.Timestamp.IsZero() {
		record.Timestamp = msg.Timestamp
	}

	for key, value := range msg.Headers {
		record.Headers = append(record.Headers, kgo.RecordHeader{
			Key:   key,
			Value: []byte(value),
		})
	}

	p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
		if callback != nil {
			callback(err)
		}
	})
}

// Flush waits for all buffered records to be sent
func (p *Producer) Flush(ctx context.Context) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	return p.client.Flush(ctx)
}

// Close closes the producer
func (p *Producer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	p.client.Close()
}

// Ping checks if the producer is connected
func (p *Producer) Ping(ctx context.Context) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}
	p.mu.RUnlock()

	return p.client.Ping(ctx)
}
