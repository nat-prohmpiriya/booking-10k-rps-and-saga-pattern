package worker

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueueRepository is a mock implementation of QueueRepository
type MockQueueRepository struct {
	mock.Mock
}

func (m *MockQueueRepository) JoinQueue(ctx context.Context, params repository.JoinQueueParams) (*repository.JoinQueueResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.JoinQueueResult), args.Error(1)
}

func (m *MockQueueRepository) GetPosition(ctx context.Context, eventID, userID string) (*repository.QueuePositionResult, error) {
	args := m.Called(ctx, eventID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.QueuePositionResult), args.Error(1)
}

func (m *MockQueueRepository) LeaveQueue(ctx context.Context, eventID, userID, token string) error {
	args := m.Called(ctx, eventID, userID, token)
	return args.Error(0)
}

func (m *MockQueueRepository) GetQueueSize(ctx context.Context, eventID string) (int64, error) {
	args := m.Called(ctx, eventID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueueRepository) GetUserQueueInfo(ctx context.Context, eventID, userID string) (map[string]string, error) {
	args := m.Called(ctx, eventID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockQueueRepository) StoreQueuePass(ctx context.Context, eventID, userID, queuePass string, ttl int) error {
	args := m.Called(ctx, eventID, userID, queuePass, ttl)
	return args.Error(0)
}

func (m *MockQueueRepository) ValidateQueuePass(ctx context.Context, eventID, userID, queuePass string) (bool, error) {
	args := m.Called(ctx, eventID, userID, queuePass)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockQueueRepository) DeleteQueuePass(ctx context.Context, eventID, userID string) error {
	args := m.Called(ctx, eventID, userID)
	return args.Error(0)
}

func (m *MockQueueRepository) PopUsersFromQueue(ctx context.Context, eventID string, count int64) ([]string, error) {
	args := m.Called(ctx, eventID, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockQueueRepository) GetAllQueueEventIDs(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockQueueRepository) RemoveUserFromQueue(ctx context.Context, eventID, userID string) error {
	args := m.Called(ctx, eventID, userID)
	return args.Error(0)
}

// Ensure MockQueueRepository implements QueueRepository
var _ repository.QueueRepository = (*MockQueueRepository)(nil)

func TestNewQueueReleaseWorker(t *testing.T) {
	mockRepo := new(MockQueueRepository)

	t.Run("creates worker with default config", func(t *testing.T) {
		worker := NewQueueReleaseWorker(nil, mockRepo, nil)
		assert.NotNil(t, worker)
		assert.Equal(t, 100, worker.GetBatchSize())
	})

	t.Run("creates worker with custom config", func(t *testing.T) {
		cfg := &QueueReleaseWorkerConfig{
			BatchSize:       200,
			ReleaseInterval: 5 * time.Second,
			QueuePassTTL:    10 * time.Minute,
			JWTSecret:       "custom-secret",
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)
		assert.NotNil(t, worker)
		assert.Equal(t, 200, worker.GetBatchSize())
	})

	t.Run("uses defaults for invalid config values", func(t *testing.T) {
		cfg := &QueueReleaseWorkerConfig{
			BatchSize:       -1,
			ReleaseInterval: 0,
			QueuePassTTL:    0,
			JWTSecret:       "",
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)
		assert.NotNil(t, worker)
		assert.Equal(t, 100, worker.GetBatchSize())
	})
}

func TestQueueReleaseWorker_ReleaseFromQueueOnce(t *testing.T) {
	t.Run("releases users successfully", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := &QueueReleaseWorkerConfig{
			BatchSize:    100,
			QueuePassTTL: 5 * time.Minute,
			JWTSecret:    "test-secret",
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"
		userIDs := []string{"user-1", "user-2", "user-3"}

		mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(100)).Return(userIDs, nil)
		mockRepo.On("StoreQueuePass", ctx, eventID, mock.AnythingOfType("string"), mock.AnythingOfType("string"), 300).Return(nil)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.NoError(t, err)
		assert.Len(t, releasedUsers, 3)

		for i, user := range releasedUsers {
			assert.Equal(t, userIDs[i], user.UserID)
			assert.Equal(t, eventID, user.EventID)
			assert.NotEmpty(t, user.QueuePass)
			assert.False(t, user.QueuePassExpires.IsZero())
		}

		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty when no users in queue", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := DefaultQueueReleaseWorkerConfig()
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"

		mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(100)).Return([]string{}, nil)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.NoError(t, err)
		assert.Len(t, releasedUsers, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("handles pop error gracefully", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := DefaultQueueReleaseWorkerConfig()
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"

		mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(100)).Return(nil, assert.AnError)

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.Error(t, err)
		assert.Nil(t, releasedUsers)

		mockRepo.AssertExpectations(t)
	})

	t.Run("continues releasing when store fails for some users", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		cfg := &QueueReleaseWorkerConfig{
			BatchSize:    100,
			QueuePassTTL: 5 * time.Minute,
			JWTSecret:    "test-secret",
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		ctx := context.Background()
		eventID := "event-123"
		userIDs := []string{"user-1", "user-2", "user-3"}

		mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(100)).Return(userIDs, nil)
		// First user fails, others succeed
		mockRepo.On("StoreQueuePass", ctx, eventID, "user-1", mock.AnythingOfType("string"), 300).Return(assert.AnError).Once()
		mockRepo.On("StoreQueuePass", ctx, eventID, "user-2", mock.AnythingOfType("string"), 300).Return(nil).Once()
		mockRepo.On("StoreQueuePass", ctx, eventID, "user-3", mock.AnythingOfType("string"), 300).Return(nil).Once()

		releasedUsers, err := worker.ReleaseFromQueueOnce(ctx, eventID)

		assert.NoError(t, err)
		assert.Len(t, releasedUsers, 2) // Only 2 succeeded

		mockRepo.AssertExpectations(t)
	})
}

func TestQueueReleaseWorker_GenerateQueuePass(t *testing.T) {
	t.Run("generates valid JWT", func(t *testing.T) {
		mockRepo := new(MockQueueRepository)
		secret := "test-secret-key"
		cfg := &QueueReleaseWorkerConfig{
			BatchSize:    100,
			QueuePassTTL: 5 * time.Minute,
			JWTSecret:    secret,
		}
		worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

		queuePass, expiresAt, err := worker.generateQueuePass("user-123", "event-456")

		assert.NoError(t, err)
		assert.NotEmpty(t, queuePass)
		assert.WithinDuration(t, time.Now().Add(5*time.Minute), expiresAt, time.Second)

		// Verify JWT can be parsed
		token, err := jwt.ParseWithClaims(queuePass, &QueuePassClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		assert.NoError(t, err)
		assert.True(t, token.Valid)

		claims, ok := token.Claims.(*QueuePassClaims)
		assert.True(t, ok)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "event-456", claims.EventID)
		assert.Equal(t, "queue_pass", claims.Purpose)
		assert.Equal(t, "queue-release-worker", claims.Issuer)
	})
}

func TestQueueReleaseWorker_SetBatchSize(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	worker := NewQueueReleaseWorker(nil, mockRepo, nil)

	assert.Equal(t, 100, worker.GetBatchSize())

	worker.SetBatchSize(200)
	assert.Equal(t, 200, worker.GetBatchSize())

	// Invalid batch size should not change
	worker.SetBatchSize(0)
	assert.Equal(t, 200, worker.GetBatchSize())

	worker.SetBatchSize(-10)
	assert.Equal(t, 200, worker.GetBatchSize())
}

func TestQueueReleaseWorker_GetMetrics(t *testing.T) {
	mockRepo := new(MockQueueRepository)
	cfg := &QueueReleaseWorkerConfig{
		BatchSize:    100,
		QueuePassTTL: 5 * time.Minute,
		JWTSecret:    "test-secret",
	}
	worker := NewQueueReleaseWorker(cfg, mockRepo, nil)

	// Initial metrics should be zero
	total, lastTime, lastCount := worker.GetMetrics()
	assert.Equal(t, int64(0), total)
	assert.True(t, lastTime.IsZero())
	assert.Equal(t, 0, lastCount)

	// Release some users
	ctx := context.Background()
	eventID := "event-123"
	userIDs := []string{"user-1", "user-2"}

	mockRepo.On("PopUsersFromQueue", ctx, eventID, int64(100)).Return(userIDs, nil)
	mockRepo.On("StoreQueuePass", ctx, eventID, mock.AnythingOfType("string"), mock.AnythingOfType("string"), 300).Return(nil)

	_, _ = worker.ReleaseFromQueueOnce(ctx, eventID)

	total, lastTime, lastCount = worker.GetMetrics()
	assert.Equal(t, int64(2), total)
	assert.False(t, lastTime.IsZero())
	assert.Equal(t, 2, lastCount)
}

func TestDefaultQueueReleaseWorkerConfig(t *testing.T) {
	cfg := DefaultQueueReleaseWorkerConfig()

	assert.Equal(t, 100, cfg.BatchSize)
	assert.Equal(t, 1*time.Second, cfg.ReleaseInterval)
	assert.Equal(t, 5*time.Minute, cfg.QueuePassTTL)
	assert.NotEmpty(t, cfg.JWTSecret)
}

func TestGenerateUniqueID(t *testing.T) {
	id1 := generateUniqueID()
	id2 := generateUniqueID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should be unique
	assert.Len(t, id1, 32)       // 16 bytes = 32 hex chars
}
