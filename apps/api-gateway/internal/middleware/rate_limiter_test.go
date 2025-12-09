package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLocalRateLimiter_Allow(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   time.Minute,
		EntryTTL:          time.Minute,
	}

	limiter := NewLocalRateLimiter(config)
	defer limiter.Stop()

	key := "test-ip"

	// First 5 requests should be allowed (burst)
	for i := 0; i < 5; i++ {
		if !limiter.Allow(key) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be rejected (burst exhausted)
	if limiter.Allow(key) {
		t.Error("6th request should be rejected")
	}
}

func TestLocalRateLimiter_AllowWithRemaining(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   time.Minute,
		EntryTTL:          time.Minute,
	}

	limiter := NewLocalRateLimiter(config)
	defer limiter.Stop()

	key := "test-ip"

	// Check remaining tokens
	allowed, remaining := limiter.AllowWithRemaining(key)
	if !allowed {
		t.Error("First request should be allowed")
	}
	if remaining != 4 {
		t.Errorf("Expected 4 remaining, got %f", remaining)
	}

	// Use all burst tokens
	for i := 0; i < 4; i++ {
		limiter.Allow(key)
	}

	// Next request should be rejected
	allowed, remaining = limiter.AllowWithRemaining(key)
	if allowed {
		t.Error("Request should be rejected after burst exhausted")
	}
	if remaining >= 1 {
		t.Errorf("Expected remaining < 1, got %f", remaining)
	}
}

func TestLocalRateLimiter_TokenRefill(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 1000, // 1000 tokens per second
		BurstSize:         1,
		CleanupInterval:   time.Minute,
		EntryTTL:          time.Minute,
	}

	limiter := NewLocalRateLimiter(config)
	defer limiter.Stop()

	key := "test-ip"

	// First request should be allowed
	if !limiter.Allow(key) {
		t.Error("First request should be allowed")
	}

	// Second request should be rejected
	if limiter.Allow(key) {
		t.Error("Second request should be rejected")
	}

	// Wait for refill (1ms = 1 token at 1000 tokens/second)
	time.Sleep(2 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow(key) {
		t.Error("Request after refill should be allowed")
	}
}

func TestLocalRateLimiter_GetStats(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         3,
		CleanupInterval:   time.Minute,
		EntryTTL:          time.Minute,
	}

	limiter := NewLocalRateLimiter(config)
	defer limiter.Stop()

	key := "test-ip"

	// Make 5 requests (3 allowed, 2 rejected)
	for i := 0; i < 5; i++ {
		limiter.Allow(key)
	}

	allowed, rejected := limiter.GetStats()
	if allowed != 3 {
		t.Errorf("Expected 3 allowed, got %d", allowed)
	}
	if rejected != 2 {
		t.Errorf("Expected 2 rejected, got %d", rejected)
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		// Exact match
		{"/api/v1/events", "/api/v1/events", true},
		{"/api/v1/events", "/api/v1/events/", true},
		{"/api/v1/events/", "/api/v1/events", true},

		// Wildcard * (single segment)
		{"/api/v1/events/*", "/api/v1/events/123", true},
		{"/api/v1/events/*", "/api/v1/events/test-event", true},
		{"/api/v1/events/*", "/api/v1/events", false},
		{"/api/v1/events/*", "/api/v1/events/123/zones", false},

		// Wildcard ** (multiple segments)
		{"/api/v1/**", "/api/v1/events", true},
		{"/api/v1/**", "/api/v1/events/123", true},
		{"/api/v1/**", "/api/v1/events/123/zones", true},

		// Parameter :param (Gin-style)
		{"/api/v1/events/:id", "/api/v1/events/123", true},
		{"/api/v1/events/:id", "/api/v1/events/test-event", true},
		{"/api/v1/events/:id/zones", "/api/v1/events/123/zones", true},
		{"/api/v1/events/:id", "/api/v1/events", false},

		// Mixed patterns
		{"/api/v1/bookings/*/confirm", "/api/v1/bookings/123/confirm", true},
		{"/api/v1/bookings/*/confirm", "/api/v1/bookings/abc-def/confirm", true},
		{"/api/v1/bookings/*/confirm", "/api/v1/bookings/123/cancel", false},

		// No match
		{"/api/v1/events", "/api/v1/bookings", false},
		{"/api/v2/events", "/api/v1/events", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.path, func(t *testing.T) {
			result := matchPath(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

func TestContainsMethod(t *testing.T) {
	tests := []struct {
		methods  []string
		method   string
		expected bool
	}{
		// Empty list matches all
		{[]string{}, "GET", true},
		{[]string{}, "POST", true},
		{nil, "DELETE", true},

		// Exact match
		{[]string{"GET"}, "GET", true},
		{[]string{"POST"}, "POST", true},
		{[]string{"GET", "POST"}, "GET", true},
		{[]string{"GET", "POST"}, "POST", true},

		// Case insensitive
		{[]string{"get"}, "GET", true},
		{[]string{"GET"}, "get", true},

		// No match
		{[]string{"GET"}, "POST", false},
		{[]string{"GET", "PUT"}, "POST", false},
	}

	for _, tt := range tests {
		result := containsMethod(tt.methods, tt.method)
		if result != tt.expected {
			t.Errorf("containsMethod(%v, %q) = %v, want %v", tt.methods, tt.method, result, tt.expected)
		}
	}
}

func TestPerEndpointRateLimitConfig_FindEndpointConfig(t *testing.T) {
	config := PerEndpointRateLimitConfig{
		Default: RateLimitConfig{
			RequestsPerSecond: 1000,
			BurstSize:         100,
		},
		Endpoints: []EndpointRateLimitConfig{
			{
				PathPattern:       "/api/v1/bookings",
				Methods:           []string{"POST"},
				RequestsPerSecond: 100,
				BurstSize:         20,
			},
			{
				PathPattern:       "/api/v1/events/*",
				Methods:           []string{"GET"},
				RequestsPerSecond: 2000,
				BurstSize:         200,
			},
			{
				PathPattern:       "/api/v1/auth/*",
				Methods:           []string{}, // All methods
				RequestsPerSecond: 20,
				BurstSize:         5,
			},
		},
	}

	tests := []struct {
		method      string
		path        string
		expectedRPS int
	}{
		// Specific endpoint match
		{"POST", "/api/v1/bookings", 100},
		{"GET", "/api/v1/events/123", 2000},
		{"POST", "/api/v1/auth/login", 20},
		{"GET", "/api/v1/auth/refresh", 20}, // All methods match

		// Method mismatch - fallback to default
		{"GET", "/api/v1/bookings", 1000},

		// No match - fallback to default
		{"GET", "/api/v1/users", 1000},
		{"POST", "/api/v1/unknown", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rps, _ := config.findEndpointConfig(tt.method, tt.path)
			if rps != tt.expectedRPS {
				t.Errorf("Expected RPS %d, got %d", tt.expectedRPS, rps)
			}
		})
	}
}

func TestRateLimiterMiddleware_AllowsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         5,
		CleanupInterval:   time.Minute,
		EntryTTL:          time.Minute,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(RateLimiter(config))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First request should be allowed
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check rate limit headers
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Expected X-RateLimit-Limit header")
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}
}

func TestRateLimiterMiddleware_RejectsExcessRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         2,
		CleanupInterval:   time.Minute,
		EntryTTL:          time.Minute,
	}

	_, r := gin.CreateTestContext(httptest.NewRecorder())

	r.Use(RateLimiter(config))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Make requests until rejected
	var lastCode int
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w, req)
		lastCode = w.Code

		if w.Code == http.StatusTooManyRequests {
			// Check Retry-After header
			if w.Header().Get("Retry-After") == "" {
				t.Error("Expected Retry-After header on 429 response")
			}
			break
		}
	}

	if lastCode != http.StatusTooManyRequests {
		t.Errorf("Expected status %d after exceeding rate limit, got %d",
			http.StatusTooManyRequests, lastCode)
	}
}

func TestPerEndpointRateLimiterMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := PerEndpointRateLimitConfig{
		Default: RateLimitConfig{
			RequestsPerSecond: 1000,
			BurstSize:         100,
		},
		Endpoints: []EndpointRateLimitConfig{
			{
				PathPattern:       "/api/v1/bookings",
				Methods:           []string{"POST"},
				RequestsPerSecond: 100,
				BurstSize:         2, // Small burst for testing
			},
		},
		CleanupInterval: time.Minute,
		EntryTTL:        time.Minute,
	}

	_, r := gin.CreateTestContext(httptest.NewRecorder())

	r.Use(PerEndpointRateLimiter(config))
	r.POST("/api/v1/bookings", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.GET("/api/v1/events", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// POST /api/v1/bookings has stricter limits (burst=2)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w, req)

		if i < 2 && w.Code != http.StatusOK {
			t.Errorf("Request %d to /api/v1/bookings should be allowed", i+1)
		}
		if i >= 2 && w.Code != http.StatusTooManyRequests {
			t.Errorf("Request %d to /api/v1/bookings should be rejected", i+1)
		}
	}

	// GET /api/v1/events uses default limits (burst=100)
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
		req.RemoteAddr = "192.168.1.2:12345" // Different IP
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d to /api/v1/events should be allowed (default rate limit)", i+1)
		}
	}
}

func TestGlobalRateLimiter(t *testing.T) {
	limiter := NewGlobalRateLimiter(3)

	// First 3 should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.Acquire() {
			t.Errorf("Acquire %d should succeed", i+1)
		}
	}

	if limiter.CurrentCount() != 3 {
		t.Errorf("Expected count 3, got %d", limiter.CurrentCount())
	}

	// 4th should be rejected
	if limiter.Acquire() {
		t.Error("4th acquire should fail")
	}

	// Release one
	limiter.Release()

	if limiter.CurrentCount() != 2 {
		t.Errorf("Expected count 2 after release, got %d", limiter.CurrentCount())
	}

	// Now should be allowed again
	if !limiter.Acquire() {
		t.Error("Acquire after release should succeed")
	}
}

func TestConcurrencyLimiterMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, r := gin.CreateTestContext(httptest.NewRecorder())

	r.Use(ConcurrencyLimiter(2))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Simple test - just verify it works
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDefaultPerEndpointConfig(t *testing.T) {
	config := DefaultPerEndpointConfig()

	// Check default values
	if config.Default.RequestsPerSecond != 1000 {
		t.Errorf("Expected default RPS 1000, got %d", config.Default.RequestsPerSecond)
	}
	if config.Default.BurstSize != 100 {
		t.Errorf("Expected default burst 100, got %d", config.Default.BurstSize)
	}

	// Check endpoints exist
	if len(config.Endpoints) == 0 {
		t.Error("Expected some endpoint configurations")
	}

	// Check booking endpoint has stricter limits
	for _, endpoint := range config.Endpoints {
		if endpoint.PathPattern == "/api/v1/bookings" {
			if endpoint.RequestsPerSecond >= config.Default.RequestsPerSecond {
				t.Error("Booking endpoint should have stricter limits than default")
			}
		}
	}
}
