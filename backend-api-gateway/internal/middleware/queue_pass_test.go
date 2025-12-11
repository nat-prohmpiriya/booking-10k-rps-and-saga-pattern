package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const testJWTSecret = "test-queue-pass-secret"

// generateTestQueuePass generates a test queue pass JWT token
func generateTestQueuePass(userID, eventID string, expiresAt time.Time, secret string) string {
	claims := QueuePassClaims{
		UserID:  userID,
		EventID: eventID,
		Purpose: "queue_pass",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "booking-service",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString([]byte(secret))
	return signedToken
}

// generateInvalidPurposeToken generates a token with invalid purpose
func generateInvalidPurposeToken(userID, eventID string, secret string) string {
	claims := QueuePassClaims{
		UserID:  userID,
		EventID: eventID,
		Purpose: "invalid_purpose",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString([]byte(secret))
	return signedToken
}

func setupQueuePassTestRouter(config QueuePassConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(QueuePassMiddleware(config))

	router.POST("/api/v1/bookings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "booking created"})
	})

	router.GET("/api/v1/events", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "events list"})
	})

	return router
}

func TestQueuePassValidator_ValidateToken_Success(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(5*time.Minute), testJWTSecret)

	claims, err := validator.ValidateToken(token)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "event-456", claims.EventID)
	assert.Equal(t, "queue_pass", claims.Purpose)
}

func TestQueuePassValidator_ValidateToken_EmptyToken(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	claims, err := validator.ValidateToken("")

	assert.Error(t, err)
	assert.Equal(t, ErrMissingQueuePass, err)
	assert.Nil(t, claims)
}

func TestQueuePassValidator_ValidateToken_ExpiredToken(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	// Generate expired token
	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(-5*time.Minute), testJWTSecret)

	claims, err := validator.ValidateToken(token)

	assert.Error(t, err)
	assert.Equal(t, ErrExpiredQueuePass, err)
	assert.Nil(t, claims)
}

func TestQueuePassValidator_ValidateToken_InvalidSignature(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	// Generate token with different secret
	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(5*time.Minute), "wrong-secret")

	claims, err := validator.ValidateToken(token)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidQueuePass, err)
	assert.Nil(t, claims)
}

func TestQueuePassValidator_ValidateToken_InvalidPurpose(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	token := generateInvalidPurposeToken("user-123", "event-456", testJWTSecret)

	claims, err := validator.ValidateToken(token)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidQueuePass, err)
	assert.Nil(t, claims)
}

func TestQueuePassValidator_ValidateToken_MalformedToken(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	claims, err := validator.ValidateToken("not-a-valid-jwt")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidQueuePass, err)
	assert.Nil(t, claims)
}

func TestQueuePassValidator_IsPathProtected(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	tests := []struct {
		path     string
		method   string
		expected bool
	}{
		// Protected paths with POST
		{"/api/v1/bookings", http.MethodPost, true},
		{"/api/v1/bookings/123", http.MethodPost, true},
		{"/api/v1/bookings", http.MethodPut, true},

		// Non-protected methods
		{"/api/v1/bookings", http.MethodGet, false},
		{"/api/v1/bookings/123", http.MethodGet, false},
		{"/api/v1/bookings", http.MethodDelete, false},

		// Non-protected paths
		{"/api/v1/events", http.MethodPost, false},
		{"/api/v1/users", http.MethodPost, false},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			result := validator.IsPathProtected(tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueuePassValidator_QueueMode(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	validator := NewQueuePassValidator(config)

	// Default: disabled
	assert.False(t, validator.IsQueueModeEnabled())

	// Enable
	validator.SetQueueMode(true)
	assert.True(t, validator.IsQueueModeEnabled())

	// Disable
	validator.SetQueueMode(false)
	assert.False(t, validator.IsQueueModeEnabled())
}

func TestQueuePassMiddleware_ValidToken(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	router := setupQueuePassTestRouter(config)

	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(5*time.Minute), testJWTSecret)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
	req.Header.Set(QueuePassHeader, token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQueuePassMiddleware_NoToken_QueueModeDisabled(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	router := setupQueuePassTestRouter(config)

	// Queue mode is disabled by default, so request without token should pass
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQueuePassMiddleware_NoToken_QueueModeEnabled(t *testing.T) {
	queueMode := &atomic.Bool{}
	queueMode.Store(true) // Enable queue mode

	config := QueuePassConfig{
		JWTSecret: testJWTSecret,
		ProtectedPaths: []string{
			"/api/v1/bookings",
		},
		QueueModeEnabled: queueMode,
	}

	router := setupQueuePassTestRouter(config)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "QUEUE_REQUIRED", response["error"].(map[string]interface{})["code"])
}

func TestQueuePassMiddleware_InvalidToken_QueueModeEnabled(t *testing.T) {
	queueMode := &atomic.Bool{}
	queueMode.Store(true)

	config := QueuePassConfig{
		JWTSecret: testJWTSecret,
		ProtectedPaths: []string{
			"/api/v1/bookings",
		},
		QueueModeEnabled: queueMode,
	}

	router := setupQueuePassTestRouter(config)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
	req.Header.Set(QueuePassHeader, "invalid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestQueuePassMiddleware_ExpiredToken_QueueModeEnabled(t *testing.T) {
	queueMode := &atomic.Bool{}
	queueMode.Store(true)

	config := QueuePassConfig{
		JWTSecret: testJWTSecret,
		ProtectedPaths: []string{
			"/api/v1/bookings",
		},
		QueueModeEnabled: queueMode,
	}

	router := setupQueuePassTestRouter(config)

	// Generate expired token
	expiredToken := generateTestQueuePass("user-123", "event-456", time.Now().Add(-5*time.Minute), testJWTSecret)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
	req.Header.Set(QueuePassHeader, expiredToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestQueuePassMiddleware_ValidToken_QueueModeEnabled(t *testing.T) {
	queueMode := &atomic.Bool{}
	queueMode.Store(true)

	config := QueuePassConfig{
		JWTSecret: testJWTSecret,
		ProtectedPaths: []string{
			"/api/v1/bookings",
		},
		QueueModeEnabled: queueMode,
	}

	router := setupQueuePassTestRouter(config)

	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(5*time.Minute), testJWTSecret)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
	req.Header.Set(QueuePassHeader, token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQueuePassMiddleware_NonProtectedPath_QueueModeEnabled(t *testing.T) {
	queueMode := &atomic.Bool{}
	queueMode.Store(true)

	config := QueuePassConfig{
		JWTSecret: testJWTSecret,
		ProtectedPaths: []string{
			"/api/v1/bookings",
		},
		QueueModeEnabled: queueMode,
	}

	router := setupQueuePassTestRouter(config)

	// GET /api/v1/events is not protected
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQueuePassRateLimitBypass_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultQueuePassConfig(testJWTSecret)

	router := gin.New()
	router.Use(QueuePassRateLimitBypass(config))
	router.POST("/api/v1/bookings", func(c *gin.Context) {
		bypass, exists := c.Get("bypass_rate_limit")
		c.JSON(http.StatusOK, gin.H{
			"bypass_exists": exists,
			"bypass_value":  bypass,
		})
	})

	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(5*time.Minute), testJWTSecret)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
	req.Header.Set(QueuePassHeader, token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["bypass_exists"].(bool))
	assert.True(t, response["bypass_value"].(bool))
}

func TestQueuePassRateLimitBypass_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultQueuePassConfig(testJWTSecret)

	router := gin.New()
	router.Use(QueuePassRateLimitBypass(config))
	router.POST("/api/v1/bookings", func(c *gin.Context) {
		_, exists := c.Get("bypass_rate_limit")
		c.JSON(http.StatusOK, gin.H{
			"bypass_exists": exists,
		})
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["bypass_exists"].(bool))
}

func TestPerEndpointRateLimiterWithQueuePass_Bypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rateLimitConfig := PerEndpointRateLimitConfig{
		Default: RateLimitConfig{
			RequestsPerSecond: 1,
			BurstSize:         1, // Very strict limit
		},
		CleanupInterval: time.Minute,
		EntryTTL:        time.Minute,
	}

	router := gin.New()

	// First, queue pass bypass middleware
	queuePassConfig := DefaultQueuePassConfig(testJWTSecret)
	router.Use(QueuePassRateLimitBypass(queuePassConfig))

	// Then, rate limiter with queue pass awareness
	router.Use(PerEndpointRateLimiterWithQueuePass(rateLimitConfig))

	router.POST("/api/v1/bookings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(5*time.Minute), testJWTSecret)

	// Make multiple requests with valid queue pass - should all succeed
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
		req.Header.Set(QueuePassHeader, token)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed with queue pass", i+1)
		assert.Equal(t, "queue-pass", w.Header().Get("X-RateLimit-Bypass"))
	}
}

func TestPerEndpointRateLimiterWithQueuePass_NoBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rateLimitConfig := PerEndpointRateLimitConfig{
		Default: RateLimitConfig{
			RequestsPerSecond: 1,
			BurstSize:         2, // Allow 2 requests
		},
		CleanupInterval: time.Minute,
		EntryTTL:        time.Minute,
	}

	router := gin.New()

	// First, queue pass bypass middleware
	queuePassConfig := DefaultQueuePassConfig(testJWTSecret)
	router.Use(QueuePassRateLimitBypass(queuePassConfig))

	// Then, rate limiter with queue pass awareness
	router.Use(PerEndpointRateLimiterWithQueuePass(rateLimitConfig))

	router.POST("/api/v1/bookings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Make requests without queue pass - should be rate limited
	var lastCode int
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)
		lastCode = w.Code
	}

	assert.Equal(t, http.StatusTooManyRequests, lastCode, "Should be rate limited without queue pass")
}

func TestQueueModeController(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)
	controller := NewQueueModeController(config)

	// Default: disabled
	assert.False(t, controller.IsQueueModeEnabled())

	// Enable
	controller.EnableQueueMode()
	assert.True(t, controller.IsQueueModeEnabled())

	// Disable
	controller.DisableQueueMode()
	assert.False(t, controller.IsQueueModeEnabled())
}

func TestQueueModeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultQueuePassConfig(testJWTSecret)
	controller := NewQueueModeController(config)

	router := gin.New()
	router.GET("/queue-mode", controller.QueueModeHandler())

	req, _ := http.NewRequest(http.MethodGet, "/queue-mode", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.False(t, response["data"].(map[string]interface{})["queue_mode_enabled"].(bool))
}

func TestSetQueueModeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultQueuePassConfig(testJWTSecret)
	controller := NewQueueModeController(config)

	router := gin.New()
	router.PUT("/queue-mode", controller.SetQueueModeHandler())

	// Enable queue mode
	req, _ := http.NewRequest(http.MethodPut, "/queue-mode", strings.NewReader(`{"enabled": true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.True(t, response["data"].(map[string]interface{})["queue_mode_enabled"].(bool))

	// Verify it's actually enabled
	assert.True(t, controller.IsQueueModeEnabled())
}

func TestIsQueuePassValid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultQueuePassConfig(testJWTSecret)

	router := gin.New()
	router.Use(QueuePassMiddleware(config))
	router.POST("/api/v1/bookings", func(c *gin.Context) {
		valid := IsQueuePassValid(c)
		userID, hasUserID := GetQueuePassUserID(c)
		eventID, hasEventID := GetQueuePassEventID(c)

		c.JSON(http.StatusOK, gin.H{
			"valid":       valid,
			"user_id":     userID,
			"has_user_id": hasUserID,
			"event_id":    eventID,
			"has_event_id": hasEventID,
		})
	})

	token := generateTestQueuePass("user-123", "event-456", time.Now().Add(5*time.Minute), testJWTSecret)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", nil)
	req.Header.Set(QueuePassHeader, token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["valid"].(bool))
	assert.Equal(t, "user-123", response["user_id"])
	assert.True(t, response["has_user_id"].(bool))
	assert.Equal(t, "event-456", response["event_id"])
	assert.True(t, response["has_event_id"].(bool))
}

func TestDefaultQueuePassConfig(t *testing.T) {
	config := DefaultQueuePassConfig(testJWTSecret)

	assert.Equal(t, testJWTSecret, config.JWTSecret)
	assert.NotNil(t, config.QueueModeEnabled)
	assert.False(t, config.QueueModeEnabled.Load())
	assert.True(t, config.BypassRateLimit)
	assert.Contains(t, config.ProtectedPaths, "/api/v1/bookings")
}
