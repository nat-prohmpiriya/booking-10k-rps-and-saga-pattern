// +build integration

package middleware

import (
	"context"
	"os"
	"testing"
	"time"

	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

func skipIfNoRedis(t *testing.T) *pkgredis.Client {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=true to run")
	}

	host := os.Getenv("TEST_REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	password := os.Getenv("TEST_REDIS_PASSWORD")

	ctx := context.Background()
	cfg := &pkgredis.Config{
		Host:          host,
		Port:          6379,
		Password:      password,
		DB:            1, // Use DB 1 for tests
		PoolSize:      10,
		MinIdleConns:  2,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		MaxRetries:    3,
		RetryInterval: time.Second,
	}

	client, err := pkgredis.NewClient(ctx, cfg)
	if err != nil {
		t.Skipf("Skipping integration test - Redis not available: %v", err)
	}

	return client
}

func TestRedisRateLimiter_Integration_Allow(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	config := RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		RedisClient:       redisClient,
		KeyPrefix:         "test:ratelimit:",
	}

	limiter := NewRedisRateLimiter(config)
	ctx := context.Background()
	key := "test-ip-" + time.Now().Format("150405.000000")

	// First 5 requests should be allowed (burst)
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be rejected
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("6th request should be rejected")
	}
}

func TestRedisRateLimiter_Integration_AllowWithRemaining(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	config := RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		RedisClient:       redisClient,
		KeyPrefix:         "test:ratelimit:",
	}

	limiter := NewRedisRateLimiter(config)
	ctx := context.Background()
	key := "test-ip-remaining-" + time.Now().Format("150405.000000")

	// First request
	allowed, remaining, err := limiter.AllowWithRemaining(ctx, key, 10, 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("First request should be allowed")
	}
	if remaining < 3 || remaining > 5 {
		t.Errorf("Expected remaining around 4, got %f", remaining)
	}

	// Exhaust burst
	for i := 0; i < 4; i++ {
		limiter.Allow(ctx, key)
	}

	// Should be rejected
	allowed, remaining, err = limiter.AllowWithRemaining(ctx, key, 10, 5)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("Request should be rejected after burst exhausted")
	}
	if remaining >= 1 {
		t.Errorf("Expected remaining < 1, got %f", remaining)
	}
}

func TestRedisRateLimiter_Integration_TokenRefill(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	config := RateLimitConfig{
		RequestsPerSecond: 100, // 100 tokens per second
		BurstSize:         1,
		RedisClient:       redisClient,
		KeyPrefix:         "test:ratelimit:",
	}

	limiter := NewRedisRateLimiter(config)
	ctx := context.Background()
	key := "test-ip-refill-" + time.Now().Format("150405.000000")

	// First request should be allowed
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("First request should be allowed")
	}

	// Second request should be rejected
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("Second request should be rejected")
	}

	// Wait for refill (10ms = 1 token at 100 tokens/second)
	time.Sleep(15 * time.Millisecond)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Request after refill should be allowed")
	}
}

func TestRedisRateLimiter_Integration_DistributedState(t *testing.T) {
	redisClient := skipIfNoRedis(t)
	defer redisClient.Close()

	config := RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         3,
		RedisClient:       redisClient,
		KeyPrefix:         "test:ratelimit:",
	}

	// Simulate two "instances" sharing state via Redis
	limiter1 := NewRedisRateLimiter(config)
	limiter2 := NewRedisRateLimiter(config)

	ctx := context.Background()
	key := "test-ip-distributed-" + time.Now().Format("150405.000000")

	// Make 2 requests from "instance 1"
	for i := 0; i < 2; i++ {
		allowed, err := limiter1.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Request %d from limiter1 should be allowed", i+1)
		}
	}

	// Make 1 request from "instance 2" - should see shared state
	allowed, err := limiter2.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("3rd request from limiter2 should be allowed (burst=3)")
	}

	// 4th request from either instance should be rejected
	allowed, err = limiter2.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("4th request should be rejected (burst exhausted)")
	}
}
