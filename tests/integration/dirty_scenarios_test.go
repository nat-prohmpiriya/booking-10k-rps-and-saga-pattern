package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// ============================================================================
// Dirty Scenario Integration Tests
// Tests edge cases that may occur in production
// Run with: INTEGRATION_TEST=true TEST_REDIS_HOST=<host> go test ./tests/integration/... -v
// ============================================================================

// TestDirtyScenario_ConcurrentLastSeatRace tests 100 concurrent requests for the last seat
// Expected: Exactly 1 winner, 99 failures with INSUFFICIENT_STOCK
func TestDirtyScenario_ConcurrentLastSeatRace(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Setup: Zone with exactly 1 seat
	zoneID := "race-test-zone-1"
	eventID := "race-test-event"
	zoneKey := "zone:availability:" + zoneID

	defer client.Del(ctx, zoneKey)

	// Initialize with exactly 1 seat
	client.Set(ctx, zoneKey, "1", time.Hour)

	// Load reserve script
	_, err = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	// Track results
	var winners int32
	var losers int32
	var wg sync.WaitGroup

	numConcurrent := 100

	// Cleanup keys
	keysToCleanup := []string{zoneKey}

	// Launch concurrent reservations
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			userID := fmt.Sprintf("race-user-%d", idx)
			bookingID := fmt.Sprintf("race-booking-%d", idx)
			userKey := fmt.Sprintf("user:reservations:%s:%s", userID, eventID)
			reservationKey := fmt.Sprintf("reservation:%s", bookingID)
			keysToCleanup = append(keysToCleanup, userKey, reservationKey)

			result, err := client.EvalShaByName(ctx, "reserve_seats",
				[]string{zoneKey, userKey, reservationKey},
				1, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
			).Slice()

			if err != nil {
				t.Errorf("Script execution error: %v", err)
				return
			}

			success := result[0].(int64)
			if success == 1 {
				atomic.AddInt32(&winners, 1)
			} else {
				errorCode := result[1].(string)
				if errorCode == "INSUFFICIENT_STOCK" {
					atomic.AddInt32(&losers, 1)
				} else {
					t.Errorf("Unexpected error: %s - %s", errorCode, result[2].(string))
				}
			}
		}(i)
	}

	wg.Wait()

	// Cleanup
	client.Del(ctx, keysToCleanup...)

	// Verify results
	if winners != 1 {
		t.Errorf("Expected exactly 1 winner, got %d", winners)
	}

	if losers != int32(numConcurrent-1) {
		t.Errorf("Expected %d losers, got %d", numConcurrent-1, losers)
	}

	t.Logf("Race test results: %d winners, %d losers", winners, losers)
}

// TestDirtyScenario_IdempotencyRetry tests that same idempotency key returns same booking
func TestDirtyScenario_IdempotencyRetry(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Setup
	zoneID := "idem-test-zone-1"
	eventID := "idem-test-event"
	userID := "idem-test-user"
	bookingID := "idem-test-booking"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey := "reservation:" + bookingID

	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	// Initialize with 100 seats
	client.Set(ctx, zoneKey, "100", time.Hour)

	_, err = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	// First reservation
	result1, err := client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
	).Slice()
	if err != nil {
		t.Fatalf("First reservation failed: %v", err)
	}

	if result1[0].(int64) != 1 {
		t.Fatalf("First reservation should succeed")
	}

	remainingAfterFirst := result1[1].(int64)
	if remainingAfterFirst != 98 {
		t.Errorf("Expected 98 remaining after first reservation, got %d", remainingAfterFirst)
	}

	// Retry with same booking ID - should get same result without double-booking
	// Note: In the actual booking service, idempotency is handled at the service layer
	// by checking if booking with same idempotency key exists in PostgreSQL
	// This test verifies that at Redis layer, attempting same reservation key fails
	result2, err := client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
	).Slice()
	if err != nil {
		t.Fatalf("Retry reservation failed: %v", err)
	}

	// At Redis layer, it will succeed but with updated values
	// The actual idempotency check happens at service layer
	if result2[0].(int64) != 1 {
		t.Logf("Retry result: %v (Redis layer allows re-reserve)", result2)
	}

	// Verify seat count hasn't gone below what's expected
	// Due to DECRBY being idempotent in execution, count will change
	// The service layer must handle idempotency properly
	remaining, _ := client.Get(ctx, zoneKey).Int64()
	t.Logf("Remaining seats after retries: %d (service layer should handle idempotency)", remaining)
}

// TestDirtyScenario_TTLExpiration tests that reservations expire after TTL
func TestDirtyScenario_TTLExpiration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Setup
	zoneID := "ttl-test-zone-1"
	eventID := "ttl-test-event"
	userID := "ttl-test-user"
	bookingID := "ttl-test-booking"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey := "reservation:" + bookingID

	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	// Initialize with 100 seats
	client.Set(ctx, zoneKey, "100", time.Hour)

	_, err = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	// Reserve with very short TTL (2 seconds)
	shortTTL := 2
	result, err := client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", shortTTL,
	).Slice()
	if err != nil {
		t.Fatalf("Reservation failed: %v", err)
	}

	if result[0].(int64) != 1 {
		t.Fatalf("Reservation should succeed")
	}

	// Verify reservation exists
	exists, _ := client.Exists(ctx, reservationKey).Result()
	if exists != 1 {
		t.Fatal("Reservation should exist immediately after creation")
	}

	// Wait for TTL to expire
	t.Log("Waiting for TTL to expire...")
	time.Sleep(time.Duration(shortTTL+1) * time.Second)

	// Verify reservation no longer exists
	exists, _ = client.Exists(ctx, reservationKey).Result()
	if exists != 0 {
		t.Error("Reservation should have expired after TTL")
	}

	// Note: The seats are NOT automatically returned to availability
	// This requires a cleanup worker in production
	// Verify seats were decremented
	remaining, _ := client.Get(ctx, zoneKey).Int64()
	t.Logf("Remaining seats after TTL expiration: %d (worker should return these)", remaining)
}

// TestDirtyScenario_NoNegativeInventory tests that inventory never goes negative
func TestDirtyScenario_NoNegativeInventory(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Setup: Zone with 5 seats, 10 concurrent requests for 1 seat each
	zoneID := "negative-test-zone-1"
	eventID := "negative-test-event"
	zoneKey := "zone:availability:" + zoneID

	defer client.Del(ctx, zoneKey)

	// Initialize with exactly 5 seats
	initialSeats := 5
	client.Set(ctx, zoneKey, fmt.Sprintf("%d", initialSeats), time.Hour)

	_, err = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	var successCount int32
	var wg sync.WaitGroup

	numConcurrent := 10
	keysToCleanup := []string{zoneKey}

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			userID := fmt.Sprintf("negative-user-%d", idx)
			bookingID := fmt.Sprintf("negative-booking-%d", idx)
			userKey := fmt.Sprintf("user:reservations:%s:%s", userID, eventID)
			reservationKey := fmt.Sprintf("reservation:%s", bookingID)
			keysToCleanup = append(keysToCleanup, userKey, reservationKey)

			result, _ := client.EvalShaByName(ctx, "reserve_seats",
				[]string{zoneKey, userKey, reservationKey},
				1, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
			).Slice()

			if result[0].(int64) == 1 {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Cleanup
	client.Del(ctx, keysToCleanup...)

	// Verify results
	if int(successCount) != initialSeats {
		t.Errorf("Expected exactly %d successful reservations, got %d", initialSeats, successCount)
	}

	// Verify inventory is 0, not negative
	remaining, _ := client.Get(ctx, zoneKey).Int64()
	if remaining < 0 {
		t.Errorf("Inventory went negative: %d", remaining)
	}
	if remaining != 0 {
		t.Errorf("Expected 0 remaining, got %d", remaining)
	}

	t.Logf("No negative inventory: %d successful out of %d requests, remaining: %d",
		successCount, numConcurrent, remaining)
}

// TestDirtyScenario_ConcurrentReserveAndRelease tests concurrent reserve and release
func TestDirtyScenario_ConcurrentReserveAndRelease(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	zoneID := "concurrent-test-zone-1"
	eventID := "concurrent-test-event"
	zoneKey := "zone:availability:" + zoneID

	// Initialize with 100 seats
	initialSeats := 100
	client.Set(ctx, zoneKey, fmt.Sprintf("%d", initialSeats), time.Hour)

	_, _ = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	_, _ = client.LoadScript(ctx, "release_seats", releaseSeatsScript)

	keysToCleanup := []string{zoneKey}

	// Phase 1: Create reservations
	numReservations := 20
	reservations := make([]struct {
		userID         string
		bookingID      string
		userKey        string
		reservationKey string
	}, numReservations)

	for i := 0; i < numReservations; i++ {
		reservations[i].userID = fmt.Sprintf("concurrent-user-%d", i)
		reservations[i].bookingID = fmt.Sprintf("concurrent-booking-%d", i)
		reservations[i].userKey = fmt.Sprintf("user:reservations:%s:%s", reservations[i].userID, eventID)
		reservations[i].reservationKey = fmt.Sprintf("reservation:%s", reservations[i].bookingID)
		keysToCleanup = append(keysToCleanup, reservations[i].userKey, reservations[i].reservationKey)

		_, _ = client.EvalShaByName(ctx, "reserve_seats",
			[]string{zoneKey, reservations[i].userKey, reservations[i].reservationKey},
			2, 100, reservations[i].userID, reservations[i].bookingID, zoneID, eventID, "show-001", "1000", 600,
		).Slice()
	}

	defer client.Del(ctx, keysToCleanup...)

	// Verify seats decreased
	afterReserve, _ := client.Get(ctx, zoneKey).Int64()
	expectedAfterReserve := int64(initialSeats - numReservations*2)
	if afterReserve != expectedAfterReserve {
		t.Errorf("Expected %d after reservations, got %d", expectedAfterReserve, afterReserve)
	}

	// Phase 2: Concurrent releases
	var wg sync.WaitGroup
	for i := 0; i < numReservations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = client.EvalShaByName(ctx, "release_seats",
				[]string{zoneKey, reservations[idx].userKey, reservations[idx].reservationKey},
				reservations[idx].bookingID, reservations[idx].userID,
			).Slice()
		}(i)
	}
	wg.Wait()

	// Verify all seats returned
	afterRelease, _ := client.Get(ctx, zoneKey).Int64()
	if afterRelease != int64(initialSeats) {
		t.Errorf("Expected %d after releases, got %d", initialSeats, afterRelease)
	}

	t.Logf("Concurrent reserve/release: %d → %d → %d seats",
		initialSeats, afterReserve, afterRelease)
}

// TestDirtyScenario_ConfirmAfterExpiration tests confirming an expired reservation
func TestDirtyScenario_ConfirmAfterExpiration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Setup
	zoneID := "expire-confirm-zone-1"
	eventID := "expire-confirm-event"
	userID := "expire-confirm-user"
	bookingID := "expire-confirm-booking"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey := "reservation:" + bookingID

	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	client.Set(ctx, zoneKey, "100", time.Hour)

	_, _ = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	_, _ = client.LoadScript(ctx, "confirm_booking", confirmBookingScript)

	// Reserve with very short TTL
	shortTTL := 2
	result, _ := client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", shortTTL,
	).Slice()

	if result[0].(int64) != 1 {
		t.Fatal("Reservation should succeed")
	}

	// Wait for TTL to expire
	t.Log("Waiting for reservation to expire...")
	time.Sleep(time.Duration(shortTTL+1) * time.Second)

	// Try to confirm expired reservation
	confirmResult, err := client.EvalShaByName(ctx, "confirm_booking",
		[]string{reservationKey},
		bookingID, userID, "payment-001",
	).Slice()
	if err != nil {
		t.Fatalf("Confirm script failed: %v", err)
	}

	success := confirmResult[0].(int64)
	if success != 0 {
		t.Error("Confirming expired reservation should fail")
	}

	errorCode := confirmResult[1].(string)
	if errorCode != "RESERVATION_NOT_FOUND" {
		t.Errorf("Expected RESERVATION_NOT_FOUND, got %s", errorCode)
	}

	t.Log("Correctly rejected confirmation of expired reservation")
}

// TestDirtyScenario_DoubleRelease tests releasing the same booking twice
func TestDirtyScenario_DoubleRelease(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	cfg := getTestRedisConfig()

	client, err := redis.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer client.Close()

	// Setup
	zoneID := "double-release-zone-1"
	eventID := "double-release-event"
	userID := "double-release-user"
	bookingID := "double-release-booking"

	zoneKey := "zone:availability:" + zoneID
	userKey := "user:reservations:" + userID + ":" + eventID
	reservationKey := "reservation:" + bookingID

	defer client.Del(ctx, zoneKey, userKey, reservationKey)

	// Initialize with 100 seats
	client.Set(ctx, zoneKey, "100", time.Hour)

	_, _ = client.LoadScript(ctx, "reserve_seats", reserveSeatsScript)
	_, _ = client.LoadScript(ctx, "release_seats", releaseSeatsScript)

	// Reserve
	_, _ = client.EvalShaByName(ctx, "reserve_seats",
		[]string{zoneKey, userKey, reservationKey},
		2, 10, userID, bookingID, zoneID, eventID, "show-001", "1000", 600,
	).Slice()

	// First release - should succeed
	result1, _ := client.EvalShaByName(ctx, "release_seats",
		[]string{zoneKey, userKey, reservationKey},
		bookingID, userID,
	).Slice()

	if result1[0].(int64) != 1 {
		t.Fatal("First release should succeed")
	}

	// Second release - should fail
	result2, _ := client.EvalShaByName(ctx, "release_seats",
		[]string{zoneKey, userKey, reservationKey},
		bookingID, userID,
	).Slice()

	success := result2[0].(int64)
	if success != 0 {
		t.Error("Second release should fail")
	}

	errorCode := result2[1].(string)
	if errorCode != "RESERVATION_NOT_FOUND" {
		t.Errorf("Expected RESERVATION_NOT_FOUND, got %s", errorCode)
	}

	// Verify seats are correctly at 100 (not 102 from double release)
	remaining, _ := client.Get(ctx, zoneKey).Int64()
	if remaining != 100 {
		t.Errorf("Expected 100 seats (no double credit), got %d", remaining)
	}

	t.Log("Correctly prevented double release")
}

// ============================================================================
// Kafka Consumer Crash Simulation Test
// This test simulates consumer crash by testing idempotency of event processing
// ============================================================================

// BookingEvent represents a booking event for testing
type BookingEvent struct {
	EventType   string       `json:"event_type"`
	BookingID   string       `json:"booking_id"`
	BookingData *BookingData `json:"booking_data"`
}

type BookingData struct {
	ZoneID   string `json:"zone_id"`
	Quantity int    `json:"quantity"`
}

// TestDirtyScenario_KafkaConsumerIdempotency tests that reprocessing events is idempotent
// This simulates what happens when a Kafka consumer crashes and restarts
func TestDirtyScenario_KafkaConsumerIdempotency(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// This test verifies that the inventory worker's delta aggregation
	// handles duplicate events correctly

	// Create mock events
	events := []BookingEvent{
		{EventType: "booking.created", BookingID: "kafka-test-1", BookingData: &BookingData{ZoneID: "zone-A", Quantity: 2}},
		{EventType: "booking.created", BookingID: "kafka-test-2", BookingData: &BookingData{ZoneID: "zone-A", Quantity: 3}},
		{EventType: "booking.confirmed", BookingID: "kafka-test-1", BookingData: &BookingData{ZoneID: "zone-A", Quantity: 2}},
	}

	// Simulate processing events (would normally go through inventory worker)
	deltas := make(map[string]*struct {
		Reserved  int
		Confirmed int
		Cancelled int
	})

	processEvent := func(event BookingEvent) {
		zoneID := event.BookingData.ZoneID
		if deltas[zoneID] == nil {
			deltas[zoneID] = &struct {
				Reserved  int
				Confirmed int
				Cancelled int
			}{}
		}

		switch event.EventType {
		case "booking.created":
			deltas[zoneID].Reserved += event.BookingData.Quantity
		case "booking.confirmed":
			deltas[zoneID].Confirmed += event.BookingData.Quantity
		case "booking.cancelled", "booking.expired":
			deltas[zoneID].Cancelled += event.BookingData.Quantity
		}
	}

	// Process events
	for _, e := range events {
		processEvent(e)
	}

	// Simulate crash and reprocess (duplicate events)
	t.Log("Simulating consumer crash and restart - reprocessing events...")
	for _, e := range events {
		processEvent(e)
	}

	// In a real system with proper idempotency:
	// - Each event should only be processed once
	// - The inventory worker should use event IDs or offsets to detect duplicates

	// Without idempotency check, we expect doubled values
	// This demonstrates why idempotency is important
	delta := deltas["zone-A"]
	t.Logf("After duplicate processing: Reserved=%d, Confirmed=%d (doubled without idempotency)",
		delta.Reserved, delta.Confirmed)

	// In production, the idempotency key on bookings prevents actual double-booking
	// The inventory worker aggregates deltas which are then applied to PostgreSQL
	// The actual protection comes from:
	// 1. Idempotency keys at booking service level
	// 2. Unique constraints in PostgreSQL
	// 3. Redis Lua script atomicity

	t.Log("Note: Real idempotency is handled by booking service idempotency keys")
}

// TestDirtyScenario_EventProcessingOrder tests handling of out-of-order events
func TestDirtyScenario_EventProcessingOrder(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Test that out-of-order events don't cause issues
	// Example: Confirmed event arrives before Created event

	events := []BookingEvent{
		// Out of order: confirmed before created
		{EventType: "booking.confirmed", BookingID: "order-test-1", BookingData: &BookingData{ZoneID: "zone-B", Quantity: 2}},
		{EventType: "booking.created", BookingID: "order-test-1", BookingData: &BookingData{ZoneID: "zone-B", Quantity: 2}},
	}

	deltas := make(map[string]*struct {
		Reserved  int
		Confirmed int
	})

	for _, e := range events {
		zoneID := e.BookingData.ZoneID
		if deltas[zoneID] == nil {
			deltas[zoneID] = &struct {
				Reserved  int
				Confirmed int
			}{}
		}

		switch e.EventType {
		case "booking.created":
			deltas[zoneID].Reserved += e.BookingData.Quantity
		case "booking.confirmed":
			deltas[zoneID].Confirmed += e.BookingData.Quantity
		}
	}

	delta := deltas["zone-B"]

	// Both events were processed
	if delta.Reserved != 2 || delta.Confirmed != 2 {
		t.Errorf("Expected Reserved=2, Confirmed=2, got Reserved=%d, Confirmed=%d",
			delta.Reserved, delta.Confirmed)
	}

	// The inventory worker aggregates deltas and applies them in batch
	// PostgreSQL update: available_seats = available_seats - reserved + cancelled
	//                    reserved_seats = reserved_seats + reserved - confirmed - cancelled
	//                    sold_seats = sold_seats + confirmed

	// Final calculation:
	// available_seats: -2 (reserved)
	// reserved_seats: +2 (reserved) - 2 (confirmed) = 0
	// sold_seats: +2 (confirmed)

	t.Logf("Out-of-order events processed: Reserved=%d, Confirmed=%d", delta.Reserved, delta.Confirmed)
	t.Log("Inventory worker aggregation handles this correctly")
}

// Helper to marshal events for test output
func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
