package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// Queue Pass header name
const QueuePassHeader = "X-Queue-Pass"

// Context key for queue pass validation result
const ContextKeyQueuePassValid = "queue_pass_valid"

// Queue pass errors
var (
	ErrMissingQueuePass = errors.New("missing queue pass")
	ErrInvalidQueuePass = errors.New("invalid queue pass")
	ErrExpiredQueuePass = errors.New("queue pass expired")
	ErrQueueModeActive  = errors.New("queue mode active, queue pass required")
)

// QueuePassClaims represents the claims in a queue pass JWT
type QueuePassClaims struct {
	UserID  string `json:"user_id"`
	EventID string `json:"event_id"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// QueuePassConfig holds configuration for queue pass middleware
type QueuePassConfig struct {
	// Secret key for validating queue pass JWT
	JWTSecret string
	// Redis client for validating queue pass (optional, for extra validation)
	RedisClient *pkgredis.Client
	// Paths that require queue pass during queue mode
	ProtectedPaths []string
	// Whether queue mode is enabled (when true, protected paths require queue pass)
	QueueModeEnabled *atomic.Bool
	// Rate limit bypass: if true, valid queue pass bypasses rate limiting
	BypassRateLimit bool
}

// DefaultQueuePassConfig returns sensible defaults
func DefaultQueuePassConfig(jwtSecret string) QueuePassConfig {
	queueMode := &atomic.Bool{}
	queueMode.Store(false) // Queue mode disabled by default

	return QueuePassConfig{
		JWTSecret: jwtSecret,
		ProtectedPaths: []string{
			"/api/v1/bookings",
		},
		QueueModeEnabled: queueMode,
		BypassRateLimit:  true,
	}
}

// QueuePassValidator handles queue pass validation logic
type QueuePassValidator struct {
	config QueuePassConfig
}

// NewQueuePassValidator creates a new queue pass validator
func NewQueuePassValidator(config QueuePassConfig) *QueuePassValidator {
	return &QueuePassValidator{
		config: config,
	}
}

// ValidateToken validates a queue pass JWT token
func (v *QueuePassValidator) ValidateToken(tokenString string) (*QueuePassClaims, error) {
	if tokenString == "" {
		return nil, ErrMissingQueuePass
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &QueuePassClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidQueuePass
		}
		return []byte(v.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredQueuePass
		}
		return nil, ErrInvalidQueuePass
	}

	if !token.Valid {
		return nil, ErrInvalidQueuePass
	}

	// Extract claims
	claims, ok := token.Claims.(*QueuePassClaims)
	if !ok {
		return nil, ErrInvalidQueuePass
	}

	// Validate purpose
	if claims.Purpose != "queue_pass" {
		return nil, ErrInvalidQueuePass
	}

	return claims, nil
}

// ValidateWithRedis performs additional validation against Redis
func (v *QueuePassValidator) ValidateWithRedis(ctx context.Context, claims *QueuePassClaims, tokenString string) (bool, error) {
	if v.config.RedisClient == nil {
		return true, nil // Skip Redis validation if not configured
	}

	// Check if queue pass exists in Redis
	key := "queue:pass:" + claims.EventID + ":" + claims.UserID
	storedPass, err := v.config.RedisClient.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil // Queue pass not found or expired
		}
		return false, err
	}

	return storedPass == tokenString, nil
}

// IsPathProtected checks if a path requires queue pass
func (v *QueuePassValidator) IsPathProtected(path, method string) bool {
	// Only POST/PUT methods to booking endpoints are protected
	if method != http.MethodPost && method != http.MethodPut {
		return false
	}

	for _, protectedPath := range v.config.ProtectedPaths {
		if strings.HasPrefix(path, protectedPath) {
			return true
		}
	}
	return false
}

// IsQueueModeEnabled returns whether queue mode is currently active
func (v *QueuePassValidator) IsQueueModeEnabled() bool {
	if v.config.QueueModeEnabled == nil {
		return false
	}
	return v.config.QueueModeEnabled.Load()
}

// SetQueueMode enables or disables queue mode
func (v *QueuePassValidator) SetQueueMode(enabled bool) {
	if v.config.QueueModeEnabled != nil {
		v.config.QueueModeEnabled.Store(enabled)
	}
}

// QueuePassMiddleware creates a middleware that validates queue pass tokens
func QueuePassMiddleware(config QueuePassConfig) gin.HandlerFunc {
	validator := NewQueuePassValidator(config)

	return func(c *gin.Context) {
		// Default: queue pass not valid
		c.Set(ContextKeyQueuePassValid, false)

		// Get queue pass from header
		queuePass := c.GetHeader(QueuePassHeader)

		// If queue pass is provided, validate it
		if queuePass != "" {
			claims, err := validator.ValidateToken(queuePass)
			if err == nil {
				// Validate with Redis if configured
				if validator.config.RedisClient != nil {
					valid, redisErr := validator.ValidateWithRedis(c.Request.Context(), claims, queuePass)
					if redisErr == nil && valid {
						c.Set(ContextKeyQueuePassValid, true)
						c.Set("queue_pass_user_id", claims.UserID)
						c.Set("queue_pass_event_id", claims.EventID)
					}
				} else {
					// No Redis validation, JWT is enough
					c.Set(ContextKeyQueuePassValid, true)
					c.Set("queue_pass_user_id", claims.UserID)
					c.Set("queue_pass_event_id", claims.EventID)
				}
			}
		}

		// Check if path is protected and queue mode is enabled
		if validator.IsQueueModeEnabled() && validator.IsPathProtected(c.Request.URL.Path, c.Request.Method) {
			// Queue mode is active, require queue pass for protected paths
			queuePassValid, _ := c.Get(ContextKeyQueuePassValid)
			if !queuePassValid.(bool) {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "QUEUE_REQUIRED",
						"message": "High traffic detected. Please join the queue first.",
					},
				})
				return
			}
		}

		c.Next()
	}
}

// QueuePassRateLimitBypass creates a middleware that bypasses rate limit for valid queue pass
// This should be used BEFORE the rate limiter middleware
func QueuePassRateLimitBypass(config QueuePassConfig) gin.HandlerFunc {
	validator := NewQueuePassValidator(config)

	return func(c *gin.Context) {
		// Get queue pass from header
		queuePass := c.GetHeader(QueuePassHeader)

		if queuePass != "" {
			claims, err := validator.ValidateToken(queuePass)
			if err == nil {
				// Optionally validate with Redis
				valid := true
				if validator.config.RedisClient != nil {
					valid, _ = validator.ValidateWithRedis(c.Request.Context(), claims, queuePass)
				}

				if valid {
					// Mark as bypass rate limit
					c.Set("bypass_rate_limit", true)
					c.Set(ContextKeyQueuePassValid, true)
					c.Set("queue_pass_user_id", claims.UserID)
					c.Set("queue_pass_event_id", claims.EventID)
				}
			}
		}

		c.Next()
	}
}

// PerEndpointRateLimiterWithQueuePass creates a rate limiter that respects queue pass bypass
func PerEndpointRateLimiterWithQueuePass(config PerEndpointRateLimitConfig) gin.HandlerFunc {
	// Get the base rate limiter
	baseLimiter := PerEndpointRateLimiter(config)

	return func(c *gin.Context) {
		// Check if this request has a valid queue pass and should bypass rate limit
		if bypass, exists := c.Get("bypass_rate_limit"); exists {
			if bypassBool, ok := bypass.(bool); ok && bypassBool {
				// Set headers to indicate bypass
				c.Header("X-RateLimit-Bypass", "queue-pass")
				c.Next()
				return
			}
		}

		// Apply normal rate limiting
		baseLimiter(c)
	}
}

// QueueModeController provides API for controlling queue mode
type QueueModeController struct {
	validator *QueuePassValidator
}

// NewQueueModeController creates a new queue mode controller
func NewQueueModeController(config QueuePassConfig) *QueueModeController {
	return &QueueModeController{
		validator: NewQueuePassValidator(config),
	}
}

// EnableQueueMode enables queue mode
func (c *QueueModeController) EnableQueueMode() {
	c.validator.SetQueueMode(true)
}

// DisableQueueMode disables queue mode
func (c *QueueModeController) DisableQueueMode() {
	c.validator.SetQueueMode(false)
}

// IsQueueModeEnabled returns whether queue mode is enabled
func (c *QueueModeController) IsQueueModeEnabled() bool {
	return c.validator.IsQueueModeEnabled()
}

// QueueModeHandler returns a handler for queue mode status endpoint
func (c *QueueModeController) QueueModeHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"queue_mode_enabled": c.IsQueueModeEnabled(),
				"timestamp":          time.Now().Unix(),
			},
		})
	}
}

// SetQueueModeHandler returns a handler for enabling/disabling queue mode
func (c *QueueModeController) SetQueueModeHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid request body",
				},
			})
			return
		}

		c.validator.SetQueueMode(req.Enabled)

		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"queue_mode_enabled": c.IsQueueModeEnabled(),
				"message":            "Queue mode updated successfully",
			},
		})
	}
}

// IsQueuePassValid checks if the current request has a valid queue pass
func IsQueuePassValid(c *gin.Context) bool {
	if valid, exists := c.Get(ContextKeyQueuePassValid); exists {
		if validBool, ok := valid.(bool); ok {
			return validBool
		}
	}
	return false
}

// GetQueuePassUserID gets the user ID from queue pass claims
func GetQueuePassUserID(c *gin.Context) (string, bool) {
	if userID, exists := c.Get("queue_pass_user_id"); exists {
		if id, ok := userID.(string); ok {
			return id, true
		}
	}
	return "", false
}

// GetQueuePassEventID gets the event ID from queue pass claims
func GetQueuePassEventID(c *gin.Context) (string, bool) {
	if eventID, exists := c.Get("queue_pass_event_id"); exists {
		if id, ok := eventID.(string); ok {
			return id, true
		}
	}
	return "", false
}
