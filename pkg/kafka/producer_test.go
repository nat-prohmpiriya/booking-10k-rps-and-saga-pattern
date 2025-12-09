package kafka

import (
	"context"
	"testing"
	"time"
)

func TestProducerConfig(t *testing.T) {
	t.Run("NewProducer with nil config returns error", func(t *testing.T) {
		ctx := context.Background()
		_, err := NewProducer(ctx, nil)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("NewProducer with empty brokers returns error", func(t *testing.T) {
		ctx := context.Background()
		_, err := NewProducer(ctx, &ProducerConfig{
			Brokers: []string{},
		})
		if err == nil {
			t.Error("expected error for empty brokers")
		}
	})
}

func TestMessage(t *testing.T) {
	t.Run("Message with all fields", func(t *testing.T) {
		msg := &Message{
			Topic:     "test-topic",
			Key:       []byte("test-key"),
			Value:     []byte("test-value"),
			Headers:   map[string]string{"header1": "value1"},
			Timestamp: time.Now(),
		}

		if msg.Topic != "test-topic" {
			t.Errorf("expected topic 'test-topic', got %s", msg.Topic)
		}
		if string(msg.Key) != "test-key" {
			t.Errorf("expected key 'test-key', got %s", string(msg.Key))
		}
		if string(msg.Value) != "test-value" {
			t.Errorf("expected value 'test-value', got %s", string(msg.Value))
		}
		if msg.Headers["header1"] != "value1" {
			t.Errorf("expected header 'value1', got %s", msg.Headers["header1"])
		}
	})
}

func TestProducerConfigDefaults(t *testing.T) {
	cfg := &ProducerConfig{
		Brokers:       []string{"localhost:9092"},
		ClientID:      "test-client",
		MaxRetries:    0, // Should default to 3
		RetryInterval: 0, // Should default to 2s
	}

	if cfg.MaxRetries != 0 {
		t.Errorf("expected MaxRetries 0, got %d", cfg.MaxRetries)
	}

	// When passed to NewProducer, these will be set to defaults internally
	// This test verifies the config struct allows zero values
}
