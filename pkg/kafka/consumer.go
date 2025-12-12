package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/trace"
)

// Consumer represents a Kafka consumer
type Consumer struct {
	client *kgo.Client
	mu     sync.RWMutex
	closed bool
}

// ConsumerConfig contains configuration for the Kafka consumer
type ConsumerConfig struct {
	Brokers         []string
	GroupID         string
	Topics          []string
	ClientID        string
	MaxRetries      int
	RetryInterval   time.Duration
	SessionTimeout  time.Duration
	RebalanceTimeout time.Duration
	AutoCommit      bool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(ctx context.Context, cfg *ConsumerConfig) (*Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("consumer config is required")
	}

	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}

	if cfg.GroupID == "" {
		return nil, fmt.Errorf("consumer group ID is required")
	}

	if len(cfg.Topics) == 0 {
		return nil, fmt.Errorf("at least one topic is required")
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.DisableAutoCommit(),
	}

	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}

	if cfg.SessionTimeout > 0 {
		opts = append(opts, kgo.SessionTimeout(cfg.SessionTimeout))
	}

	if cfg.RebalanceTimeout > 0 {
		opts = append(opts, kgo.RebalanceTimeout(cfg.RebalanceTimeout))
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
		return nil, fmt.Errorf("failed to create kafka consumer after %d retries: %w", maxRetries, err)
	}

	return &Consumer{
		client: client,
	}, nil
}

// Poll fetches records from Kafka
func (c *Consumer) Poll(ctx context.Context) ([]*Record, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	fetches := c.client.PollFetches(ctx)
	if errs := fetches.Errors(); len(errs) > 0 {
		// Return first error
		for _, err := range errs {
			return nil, fmt.Errorf("poll error on topic %s partition %d: %w", err.Topic, err.Partition, err.Err)
		}
	}

	var records []*Record
	fetches.EachRecord(func(r *kgo.Record) {
		headers := make(map[string]string)
		for _, h := range r.Headers {
			headers[h.Key] = string(h.Value)
		}

		records = append(records, &Record{
			Topic:     r.Topic,
			Partition: r.Partition,
			Offset:    r.Offset,
			Key:       r.Key,
			Value:     r.Value,
			Headers:   headers,
			Timestamp: r.Timestamp,
		})
	})

	return records, nil
}

// Record represents a consumed Kafka record
type Record struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Timestamp time.Time
}

// ExtractContext extracts trace context from record headers
func (r *Record) ExtractContext(ctx context.Context) context.Context {
	return telemetry.ExtractKafkaContext(ctx, r.Headers)
}

// StartProcessingSpan starts a consumer span for processing this record
func (r *Record) StartProcessingSpan(ctx context.Context) (context.Context, trace.Span) {
	// First extract parent context from headers
	ctx = r.ExtractContext(ctx)
	// Then start consumer span
	return telemetry.StartConsumerSpan(ctx, r.Topic, r.Partition, r.Offset, string(r.Key))
}

// CommitRecords commits the offsets for the given records
func (c *Consumer) CommitRecords(ctx context.Context, records []*Record) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	if len(records) == 0 {
		return nil
	}

	// Convert to kgo records for commit
	kgoRecords := make([]*kgo.Record, len(records))
	for i, r := range records {
		kgoRecords[i] = &kgo.Record{
			Topic:     r.Topic,
			Partition: r.Partition,
			Offset:    r.Offset,
		}
	}

	return c.client.CommitRecords(ctx, kgoRecords...)
}

// CommitOffsets commits the current offsets
func (c *Consumer) CommitOffsets(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	return c.client.CommitUncommittedOffsets(ctx)
}

// Close closes the consumer
func (c *Consumer) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	c.client.Close()
}

// Ping checks if the consumer is connected
func (c *Consumer) Ping(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("consumer is closed")
	}
	c.mu.RUnlock()

	return c.client.Ping(ctx)
}
