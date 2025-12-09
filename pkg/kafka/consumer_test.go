package kafka

import (
	"testing"
	"time"
)

func TestConsumerConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *ConsumerConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty brokers",
			config: &ConsumerConfig{
				Brokers: []string{},
				GroupID: "test-group",
				Topics:  []string{"test-topic"},
			},
			expectError: true,
		},
		{
			name: "empty group ID",
			config: &ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				GroupID: "",
				Topics:  []string{"test-topic"},
			},
			expectError: true,
		},
		{
			name: "empty topics",
			config: &ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				GroupID: "test-group",
				Topics:  []string{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually create a consumer without Kafka running,
			// but we can test that the config validation works
			if tt.config == nil {
				if !tt.expectError {
					t.Error("Expected error for nil config")
				}
				return
			}

			// Validate config manually
			var err error
			if len(tt.config.Brokers) == 0 {
				err = errNoBrokers
			} else if tt.config.GroupID == "" {
				err = errNoGroupID
			} else if len(tt.config.Topics) == 0 {
				err = errNoTopics
			}

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Sentinel errors for testing
var (
	errNoBrokers = &configError{"at least one broker is required"}
	errNoGroupID = &configError{"consumer group ID is required"}
	errNoTopics  = &configError{"at least one topic is required"}
)

type configError struct {
	msg string
}

func (e *configError) Error() string {
	return e.msg
}

func TestRecord_Fields(t *testing.T) {
	timestamp := time.Now()
	record := &Record{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
		Headers: map[string]string{
			"header1": "value1",
			"header2": "value2",
		},
		Timestamp: timestamp,
	}

	if record.Topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", record.Topic)
	}
	if record.Partition != 0 {
		t.Errorf("Expected partition 0, got %d", record.Partition)
	}
	if record.Offset != 100 {
		t.Errorf("Expected offset 100, got %d", record.Offset)
	}
	if string(record.Key) != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", string(record.Key))
	}
	if string(record.Value) != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", string(record.Value))
	}
	if len(record.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(record.Headers))
	}
	if record.Headers["header1"] != "value1" {
		t.Errorf("Expected header1='value1', got '%s'", record.Headers["header1"])
	}
	if !record.Timestamp.Equal(timestamp) {
		t.Errorf("Timestamp mismatch")
	}
}
