package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// MockRedisClient implements RedisClient interface for testing
type MockRedisClient struct {
	data map[string]string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
	}
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if val, ok := m.data[key]; ok {
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	m.data[key] = value.(string)
	cmd.SetVal("OK")
	return cmd
}

func (m *MockRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	if _, ok := m.data[key]; ok {
		cmd.SetVal(false)
	} else {
		m.data[key] = value.(string)
		cmd.SetVal(true)
	}
	return cmd
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	count := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			count++
		}
	}
	cmd.SetVal(count)
	return cmd
}

func (m *MockRedisClient) Clear() {
	m.data = make(map[string]string)
}

func setupIdempotencyTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestIdempotencyMiddleware_FirstRequest(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	router := setupIdempotencyTestRouter()
	router.POST("/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(IdempotencyKeyHeader, "unique-key-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["message"] != "success" {
		t.Errorf("Expected message 'success', got %s", response["message"])
	}
}

func TestIdempotencyMiddleware_DuplicateRequest(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	requestCount := 0
	router := setupIdempotencyTestRouter()
	router.POST("/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		requestCount++
		c.JSON(http.StatusOK, gin.H{"message": "success", "count": requestCount})
	})

	// First request
	req1, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value"}`)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(IdempotencyKeyHeader, "duplicate-key-123")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request: Expected status 200, got %d", w1.Code)
	}

	// Second request with same key
	req2, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value"}`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(IdempotencyKeyHeader, "duplicate-key-123")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request: Expected status 200, got %d", w2.Code)
	}

	// Handler should only be called once
	if requestCount != 1 {
		t.Errorf("Handler should be called only once, but was called %d times", requestCount)
	}

	// Both responses should be the same
	if w1.Body.String() != w2.Body.String() {
		t.Errorf("Responses should be identical. First: %s, Second: %s", w1.Body.String(), w2.Body.String())
	}
}

func TestIdempotencyMiddleware_MissingKey(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	router := setupIdempotencyTestRouter()
	router.POST("/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value"}`)))
	req.Header.Set("Content-Type", "application/json")
	// No idempotency key header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestIdempotencyMiddleware_DifferentBodySameKey(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	router := setupIdempotencyTestRouter()
	router.POST("/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request
	req1, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value1"}`)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(IdempotencyKeyHeader, "same-key-different-body")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request: Expected status 200, got %d", w1.Code)
	}

	// Second request with same key but different body
	req2, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value2"}`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(IdempotencyKeyHeader, "same-key-different-body")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Should return 422 - key reused with different request
	if w2.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", w2.Code)
	}
}

func TestIdempotencyMiddleware_GetRequestSkipped(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	requestCount := 0
	router := setupIdempotencyTestRouter()
	router.GET("/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		requestCount++
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// GET requests should not require idempotency key
	req, _ := http.NewRequest("GET", "/test", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Second GET request should also succeed
	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	// Handler should be called twice for GET requests
	if requestCount != 2 {
		t.Errorf("Handler should be called twice for GET, but was called %d times", requestCount)
	}
}

func TestIdempotencyMiddleware_SkipPath(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)
	config.SkipPaths = []string{"/health", "/api/*"}

	router := setupIdempotencyTestRouter()
	router.POST("/health", IdempotencyMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/api/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Health endpoint should skip idempotency
	req1, _ := http.NewRequest("POST", "/health", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Health endpoint: Expected status 200, got %d", w1.Code)
	}

	// API endpoint with wildcard should skip idempotency
	req2, _ := http.NewRequest("POST", "/api/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("API endpoint: Expected status 200, got %d", w2.Code)
	}
}

func TestIdempotencyRecord_Serialization(t *testing.T) {
	now := time.Now()
	record := &IdempotencyRecord{
		Key:          "test-key",
		Status:       StatusCompleted,
		RequestHash:  "abc123",
		ResponseCode: 200,
		ResponseBody: `{"message":"success"}`,
		CreatedAt:    now,
		CompletedAt:  &now,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Failed to marshal record: %v", err)
	}

	var decoded IdempotencyRecord
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal record: %v", err)
	}

	if decoded.Key != record.Key {
		t.Errorf("Key mismatch: expected %s, got %s", record.Key, decoded.Key)
	}
	if decoded.Status != record.Status {
		t.Errorf("Status mismatch: expected %s, got %s", record.Status, decoded.Status)
	}
	if decoded.ResponseCode != record.ResponseCode {
		t.Errorf("ResponseCode mismatch: expected %d, got %d", record.ResponseCode, decoded.ResponseCode)
	}
}

func TestDefaultKeyExtractor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("POST", "/test", nil)
	c.Request.Header.Set(IdempotencyKeyHeader, "my-idempotency-key")

	key := defaultKeyExtractor(c)
	if key != "my-idempotency-key" {
		t.Errorf("Expected 'my-idempotency-key', got '%s'", key)
	}
}

func TestRequireIdempotencyKey_Present(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", RequireIdempotencyKey(), func(c *gin.Context) {
		key, ok := GetIdempotencyKey(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "key not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": key})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Set(IdempotencyKeyHeader, "test-key-123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["key"] != "test-key-123" {
		t.Errorf("Expected key 'test-key-123', got '%s'", response["key"])
	}
}

func TestRequireIdempotencyKey_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", RequireIdempotencyKey(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	// No idempotency key

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		path     string
		pattern  string
		expected bool
	}{
		{"/health", "/health", true},
		{"/health", "/ready", false},
		{"/api/v1/users", "/api/*", true},
		{"/api/v1/users", "/api/v1/*", true},
		{"/api/v2/users", "/api/v1/*", false},
		{"/health/check", "/health*", true},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			result := matchPath(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchPath(%s, %s) = %v, expected %v", tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestGenerateRequestHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("POST", "/api/v1/test", nil)
	c.Set(ContextKeyUserID, "user-123")

	config := &IdempotencyConfig{
		IncludePathInHash: true,
		IncludeUserInHash: true,
		IncludeBodyInHash: true,
	}

	body := []byte(`{"key":"value"}`)
	hash1 := generateRequestHash(c, body, config)

	// Same request should generate same hash
	hash2 := generateRequestHash(c, body, config)
	if hash1 != hash2 {
		t.Error("Same request should generate same hash")
	}

	// Different body should generate different hash
	differentBody := []byte(`{"key":"different"}`)
	hash3 := generateRequestHash(c, differentBody, config)
	if hash1 == hash3 {
		t.Error("Different body should generate different hash")
	}
}

func TestCheckIdempotency(t *testing.T) {
	mockRedis := NewMockRedisClient()

	// Check non-existent key
	record, err := CheckIdempotency(context.Background(), mockRedis, "non-existent")
	if record != nil {
		t.Error("Expected nil record for non-existent key")
	}
	if err == nil {
		t.Error("Expected error for non-existent key")
	}

	// Store a record
	storedRecord := &IdempotencyRecord{
		Key:         "test-key",
		Status:      StatusCompleted,
		RequestHash: "hash123",
	}
	data, _ := json.Marshal(storedRecord)
	mockRedis.Set(context.Background(), IdempotencyKeyPrefix+"test-key", string(data), time.Hour)

	// Check existing key
	record, err = CheckIdempotency(context.Background(), mockRedis, "test-key")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if record == nil {
		t.Error("Expected record for existing key")
	}
	if record.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", record.Key)
	}
}

func TestDeleteIdempotencyRecord(t *testing.T) {
	mockRedis := NewMockRedisClient()

	// Store a record
	mockRedis.Set(context.Background(), IdempotencyKeyPrefix+"to-delete", "data", time.Hour)

	// Verify it exists
	result := mockRedis.Get(context.Background(), IdempotencyKeyPrefix+"to-delete")
	if result.Err() != nil {
		t.Error("Record should exist before deletion")
	}

	// Delete it
	err := DeleteIdempotencyRecord(context.Background(), mockRedis, "to-delete")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Verify it's gone
	result = mockRedis.Get(context.Background(), IdempotencyKeyPrefix+"to-delete")
	if result.Err() != redis.Nil {
		t.Error("Record should not exist after deletion")
	}
}

func TestIdempotencyMiddleware_CustomKeyExtractor(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	// Custom key extractor from request body
	config.KeyExtractor = func(c *gin.Context) string {
		return c.GetHeader("X-Request-ID")
	}

	router := setupIdempotencyTestRouter()
	router.POST("/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "custom-key-from-header")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestIdempotencyMiddleware_PutMethod(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	requestCount := 0
	router := setupIdempotencyTestRouter()
	router.PUT("/test/:id", IdempotencyMiddleware(config), func(c *gin.Context) {
		requestCount++
		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	})

	// First PUT request
	req1, _ := http.NewRequest("PUT", "/test/123", bytes.NewBuffer([]byte(`{"name":"test"}`)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(IdempotencyKeyHeader, "put-key-123")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First PUT: Expected status 200, got %d", w1.Code)
	}

	// Duplicate PUT request
	req2, _ := http.NewRequest("PUT", "/test/123", bytes.NewBuffer([]byte(`{"name":"test"}`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(IdempotencyKeyHeader, "put-key-123")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Duplicate PUT: Expected status 200, got %d", w2.Code)
	}

	if requestCount != 1 {
		t.Errorf("Handler should be called only once, but was called %d times", requestCount)
	}
}

func TestIdempotencyMiddleware_DeleteMethod(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	requestCount := 0
	router := setupIdempotencyTestRouter()
	router.DELETE("/test/:id", IdempotencyMiddleware(config), func(c *gin.Context) {
		requestCount++
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})

	// First DELETE request
	req1, _ := http.NewRequest("DELETE", "/test/123", nil)
	req1.Header.Set(IdempotencyKeyHeader, "delete-key-123")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First DELETE: Expected status 200, got %d", w1.Code)
	}

	// Duplicate DELETE request
	req2, _ := http.NewRequest("DELETE", "/test/123", nil)
	req2.Header.Set(IdempotencyKeyHeader, "delete-key-123")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Duplicate DELETE: Expected status 200, got %d", w2.Code)
	}

	if requestCount != 1 {
		t.Errorf("Handler should be called only once, but was called %d times", requestCount)
	}
}

func TestIdempotencyMiddleware_DifferentErrorResponses(t *testing.T) {
	mockRedis := NewMockRedisClient()
	config := DefaultIdempotencyConfig(mockRedis)

	router := setupIdempotencyTestRouter()
	router.POST("/test", IdempotencyMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed"})
	})

	// First request returns error
	req1, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{}`)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(IdempotencyKeyHeader, "error-key-123")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusBadRequest {
		t.Errorf("First request: Expected status 400, got %d", w1.Code)
	}

	// Second request should return cached error
	req2, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{}`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(IdempotencyKeyHeader, "error-key-123")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Cached response should return same error code
	if w2.Code != http.StatusBadRequest {
		t.Errorf("Cached request: Expected status 400, got %d", w2.Code)
	}

	if w1.Body.String() != w2.Body.String() {
		t.Errorf("Responses should be identical. First: %s, Second: %s", w1.Body.String(), w2.Body.String())
	}
}
