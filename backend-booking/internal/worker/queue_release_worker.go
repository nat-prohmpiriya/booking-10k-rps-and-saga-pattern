package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
)

// QueueReleaseWorkerConfig holds configuration for the queue release worker
type QueueReleaseWorkerConfig struct {
	// BatchSize is the number of users to release per batch (default: 100)
	BatchSize int
	// ReleaseInterval is the time between release batches (default: 1 second)
	ReleaseInterval time.Duration
	// QueuePassTTL is the TTL for queue pass tokens (default: 5 minutes)
	QueuePassTTL time.Duration
	// JWTSecret is the secret for signing queue pass JWTs
	JWTSecret string
}

// DefaultQueueReleaseWorkerConfig returns default configuration
func DefaultQueueReleaseWorkerConfig() *QueueReleaseWorkerConfig {
	return &QueueReleaseWorkerConfig{
		BatchSize:       100,
		ReleaseInterval: 1 * time.Second,
		QueuePassTTL:    5 * time.Minute,
		JWTSecret:       "queue-pass-secret-key",
	}
}

// ReleasedUser represents a user that has been released from the queue
type ReleasedUser struct {
	UserID           string
	EventID          string
	QueuePass        string
	QueuePassExpires time.Time
}

// QueueReleaseWorker releases users from the virtual queue in batches
type QueueReleaseWorker struct {
	config    *QueueReleaseWorkerConfig
	queueRepo repository.QueueRepository
	log       *logger.Logger

	// Metrics
	mu               sync.Mutex
	totalReleased    int64
	lastReleaseTime  time.Time
	lastReleaseCount int
}

// NewQueueReleaseWorker creates a new queue release worker
func NewQueueReleaseWorker(
	cfg *QueueReleaseWorkerConfig,
	queueRepo repository.QueueRepository,
	log *logger.Logger,
) *QueueReleaseWorker {
	if cfg == nil {
		cfg = DefaultQueueReleaseWorkerConfig()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.ReleaseInterval <= 0 {
		cfg.ReleaseInterval = 1 * time.Second
	}
	if cfg.QueuePassTTL <= 0 {
		cfg.QueuePassTTL = 5 * time.Minute
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "queue-pass-secret-key"
	}

	return &QueueReleaseWorker{
		config:    cfg,
		queueRepo: queueRepo,
		log:       log,
	}
}

// Start begins the continuous queue release process
func (w *QueueReleaseWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.config.ReleaseInterval)
	defer ticker.Stop()

	w.log.Info(fmt.Sprintf("Queue release worker started (batch size: %d, interval: %v)",
		w.config.BatchSize, w.config.ReleaseInterval))

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Queue release worker stopping...")
			return
		case <-ticker.C:
			w.processAllQueues(ctx)
		}
	}
}

// processAllQueues processes all active event queues
func (w *QueueReleaseWorker) processAllQueues(ctx context.Context) {
	// Get all event IDs with active queues
	eventIDs, err := w.queueRepo.GetAllQueueEventIDs(ctx)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to get queue event IDs: %v", err))
		return
	}

	if len(eventIDs) == 0 {
		return
	}

	// Process each queue
	for _, eventID := range eventIDs {
		select {
		case <-ctx.Done():
			return
		default:
			w.releaseFromQueue(ctx, eventID)
		}
	}
}

// releaseFromQueue releases a batch of users from a specific event queue
func (w *QueueReleaseWorker) releaseFromQueue(ctx context.Context, eventID string) {
	// Pop users from queue
	userIDs, err := w.queueRepo.PopUsersFromQueue(ctx, eventID, int64(w.config.BatchSize))
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to pop users from queue %s: %v", eventID, err))
		return
	}

	if len(userIDs) == 0 {
		return
	}

	w.log.Info(fmt.Sprintf("Releasing %d users from queue %s", len(userIDs), eventID))

	// Generate and store queue passes for each user
	releasedCount := 0
	for _, userID := range userIDs {
		queuePass, expiresAt, err := w.generateQueuePass(userID, eventID)
		if err != nil {
			w.log.Error(fmt.Sprintf("Failed to generate queue pass for user %s: %v", userID, err))
			continue
		}

		// Store queue pass in Redis
		ttlSeconds := int(w.config.QueuePassTTL.Seconds())
		if err := w.queueRepo.StoreQueuePass(ctx, eventID, userID, queuePass, ttlSeconds); err != nil {
			w.log.Error(fmt.Sprintf("Failed to store queue pass for user %s: %v", userID, err))
			continue
		}

		releasedCount++
		w.log.Debug(fmt.Sprintf("Released user %s from queue %s with pass expiring at %v",
			userID, eventID, expiresAt))
	}

	// Update metrics
	w.mu.Lock()
	w.totalReleased += int64(releasedCount)
	w.lastReleaseTime = time.Now()
	w.lastReleaseCount = releasedCount
	w.mu.Unlock()

	if releasedCount > 0 {
		w.log.Info(fmt.Sprintf("Successfully released %d/%d users from queue %s",
			releasedCount, len(userIDs), eventID))
	}
}

// QueuePassClaims represents the claims for a queue pass JWT
type QueuePassClaims struct {
	UserID  string `json:"user_id"`
	EventID string `json:"event_id"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// generateQueuePass generates a signed JWT queue pass token
func (w *QueueReleaseWorker) generateQueuePass(userID, eventID string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(w.config.QueuePassTTL)

	claims := QueuePassClaims{
		UserID:  userID,
		EventID: eventID,
		Purpose: "queue_pass",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "queue-release-worker",
			Subject:   userID,
			ID:        generateUniqueID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(w.config.JWTSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign queue pass: %w", err)
	}

	return signedToken, expiresAt, nil
}

// generateUniqueID generates a unique ID for JWT
func generateUniqueID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:32]
	}
	return hex.EncodeToString(bytes)
}

// GetMetrics returns current worker metrics
func (w *QueueReleaseWorker) GetMetrics() (totalReleased int64, lastReleaseTime time.Time, lastReleaseCount int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.totalReleased, w.lastReleaseTime, w.lastReleaseCount
}

// ReleaseFromQueueOnce releases a single batch from a specific queue (for testing)
func (w *QueueReleaseWorker) ReleaseFromQueueOnce(ctx context.Context, eventID string) ([]ReleasedUser, error) {
	// Pop users from queue
	userIDs, err := w.queueRepo.PopUsersFromQueue(ctx, eventID, int64(w.config.BatchSize))
	if err != nil {
		return nil, fmt.Errorf("failed to pop users from queue: %w", err)
	}

	if len(userIDs) == 0 {
		return []ReleasedUser{}, nil
	}

	var releasedUsers []ReleasedUser
	for _, userID := range userIDs {
		queuePass, expiresAt, err := w.generateQueuePass(userID, eventID)
		if err != nil {
			continue
		}

		ttlSeconds := int(w.config.QueuePassTTL.Seconds())
		if err := w.queueRepo.StoreQueuePass(ctx, eventID, userID, queuePass, ttlSeconds); err != nil {
			continue
		}

		releasedUsers = append(releasedUsers, ReleasedUser{
			UserID:           userID,
			EventID:          eventID,
			QueuePass:        queuePass,
			QueuePassExpires: expiresAt,
		})
	}

	// Update metrics
	w.mu.Lock()
	w.totalReleased += int64(len(releasedUsers))
	w.lastReleaseTime = time.Now()
	w.lastReleaseCount = len(releasedUsers)
	w.mu.Unlock()

	return releasedUsers, nil
}

// SetBatchSize updates the batch size (for dynamic configuration)
func (w *QueueReleaseWorker) SetBatchSize(size int) {
	if size > 0 {
		w.config.BatchSize = size
	}
}

// GetBatchSize returns the current batch size
func (w *QueueReleaseWorker) GetBatchSize() int {
	return w.config.BatchSize
}
