package domain

import (
	"testing"
	"time"
)

func TestNewReservation(t *testing.T) {
	r := NewReservation(
		"booking-123",
		"user-456",
		"event-789",
		"zone-abc",
		2,
		50.00,
		10*time.Minute,
	)

	if r.BookingID != "booking-123" {
		t.Errorf("BookingID = %v, want %v", r.BookingID, "booking-123")
	}
	if r.UserID != "user-456" {
		t.Errorf("UserID = %v, want %v", r.UserID, "user-456")
	}
	if r.EventID != "event-789" {
		t.Errorf("EventID = %v, want %v", r.EventID, "event-789")
	}
	if r.ZoneID != "zone-abc" {
		t.Errorf("ZoneID = %v, want %v", r.ZoneID, "zone-abc")
	}
	if r.Quantity != 2 {
		t.Errorf("Quantity = %v, want %v", r.Quantity, 2)
	}
	if r.UnitPrice != 50.00 {
		t.Errorf("UnitPrice = %v, want %v", r.UnitPrice, 50.00)
	}
	if r.TotalPrice != 100.00 {
		t.Errorf("TotalPrice = %v, want %v", r.TotalPrice, 100.00)
	}
	if r.Status != ReservationStatusReserved {
		t.Errorf("Status = %v, want %v", r.Status, ReservationStatusReserved)
	}

	// Check TTL is roughly 10 minutes
	ttl := r.TimeUntilExpiry()
	if ttl < 9*time.Minute || ttl > 10*time.Minute {
		t.Errorf("TTL = %v, expected around 10 minutes", ttl)
	}
}

func newValidReservation() *Reservation {
	return NewReservation(
		"booking-123",
		"user-456",
		"event-789",
		"zone-abc",
		2,
		50.00,
		10*time.Minute,
	)
}

func TestReservation_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Reservation)
		wantErr error
	}{
		{
			name:    "valid reservation",
			modify:  func(r *Reservation) {},
			wantErr: nil,
		},
		{
			name:    "empty BookingID",
			modify:  func(r *Reservation) { r.BookingID = "" },
			wantErr: ErrInvalidBookingID,
		},
		{
			name:    "whitespace BookingID",
			modify:  func(r *Reservation) { r.BookingID = "   " },
			wantErr: ErrInvalidBookingID,
		},
		{
			name:    "empty UserID",
			modify:  func(r *Reservation) { r.UserID = "" },
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "empty EventID",
			modify:  func(r *Reservation) { r.EventID = "" },
			wantErr: ErrInvalidEventID,
		},
		{
			name:    "empty ZoneID",
			modify:  func(r *Reservation) { r.ZoneID = "" },
			wantErr: ErrInvalidZoneID,
		},
		{
			name:    "zero quantity",
			modify:  func(r *Reservation) { r.Quantity = 0 },
			wantErr: ErrInvalidQuantity,
		},
		{
			name:    "negative quantity",
			modify:  func(r *Reservation) { r.Quantity = -1 },
			wantErr: ErrInvalidQuantity,
		},
		{
			name:    "negative unit price",
			modify:  func(r *Reservation) { r.UnitPrice = -10.00 },
			wantErr: ErrInvalidUnitPrice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newValidReservation()
			tt.modify(r)
			err := r.Validate()
			if err != tt.wantErr {
				t.Errorf("Reservation.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReservation_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(10 * time.Minute),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-10 * time.Minute),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newValidReservation()
			r.ExpiresAt = tt.expiresAt
			if got := r.IsExpired(); got != tt.want {
				t.Errorf("Reservation.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReservation_IsExpiredAt(t *testing.T) {
	r := newValidReservation()
	r.ExpiresAt = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{
			name: "before expiry",
			at:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "after expiry",
			at:   time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.IsExpiredAt(tt.at); got != tt.want {
				t.Errorf("Reservation.IsExpiredAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReservation_StatusChecks(t *testing.T) {
	tests := []struct {
		status      string
		isReserved  bool
		isConfirmed bool
		isReleased  bool
	}{
		{ReservationStatusReserved, true, false, false},
		{ReservationStatusConfirmed, false, true, false},
		{ReservationStatusReleased, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			r := newValidReservation()
			r.Status = tt.status

			if got := r.IsReserved(); got != tt.isReserved {
				t.Errorf("Reservation.IsReserved() = %v, want %v", got, tt.isReserved)
			}
			if got := r.IsConfirmed(); got != tt.isConfirmed {
				t.Errorf("Reservation.IsConfirmed() = %v, want %v", got, tt.isConfirmed)
			}
			if got := r.IsReleased(); got != tt.isReleased {
				t.Errorf("Reservation.IsReleased() = %v, want %v", got, tt.isReleased)
			}
		})
	}
}

func TestReservation_CanConfirm(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Reservation)
		want   bool
	}{
		{
			name:   "reserved and not expired",
			modify: func(r *Reservation) {},
			want:   true,
		},
		{
			name: "reserved but expired",
			modify: func(r *Reservation) {
				r.ExpiresAt = time.Now().Add(-1 * time.Minute)
			},
			want: false,
		},
		{
			name:   "already confirmed",
			modify: func(r *Reservation) { r.Status = ReservationStatusConfirmed },
			want:   false,
		},
		{
			name:   "released",
			modify: func(r *Reservation) { r.Status = ReservationStatusReleased },
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newValidReservation()
			tt.modify(r)
			if got := r.CanConfirm(); got != tt.want {
				t.Errorf("Reservation.CanConfirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReservation_CanRelease(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"reserved", ReservationStatusReserved, true},
		{"confirmed", ReservationStatusConfirmed, false},
		{"released", ReservationStatusReleased, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newValidReservation()
			r.Status = tt.status
			if got := r.CanRelease(); got != tt.want {
				t.Errorf("Reservation.CanRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReservation_TTL(t *testing.T) {
	t.Run("positive TTL", func(t *testing.T) {
		r := newValidReservation()
		r.ExpiresAt = time.Now().Add(5 * time.Minute)

		ttl := r.TTL()
		if ttl < 250 || ttl > 300 {
			t.Errorf("TTL() = %v, expected around 300 seconds", ttl)
		}
	})

	t.Run("expired returns 0", func(t *testing.T) {
		r := newValidReservation()
		r.ExpiresAt = time.Now().Add(-1 * time.Minute)

		if ttl := r.TTL(); ttl != 0 {
			t.Errorf("TTL() = %v, want 0", ttl)
		}
	})
}

func TestReservation_BelongsToUser(t *testing.T) {
	r := newValidReservation()
	r.UserID = "user-123"

	if !r.BelongsToUser("user-123") {
		t.Error("BelongsToUser() should return true for matching user")
	}
	if r.BelongsToUser("user-456") {
		t.Error("BelongsToUser() should return false for non-matching user")
	}
}

func TestReservation_JSON(t *testing.T) {
	r := newValidReservation()

	// Test ToJSON
	jsonStr, err := r.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	if jsonStr == "" {
		t.Error("ToJSON() returned empty string")
	}

	// Test ReservationFromJSON
	parsed, err := ReservationFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("ReservationFromJSON() error = %v", err)
	}

	if parsed.BookingID != r.BookingID {
		t.Errorf("BookingID = %v, want %v", parsed.BookingID, r.BookingID)
	}
	if parsed.UserID != r.UserID {
		t.Errorf("UserID = %v, want %v", parsed.UserID, r.UserID)
	}
	if parsed.Quantity != r.Quantity {
		t.Errorf("Quantity = %v, want %v", parsed.Quantity, r.Quantity)
	}
	if parsed.TotalPrice != r.TotalPrice {
		t.Errorf("TotalPrice = %v, want %v", parsed.TotalPrice, r.TotalPrice)
	}
}

func TestReservationFromJSON_Invalid(t *testing.T) {
	_, err := ReservationFromJSON("invalid json")
	if err == nil {
		t.Error("ReservationFromJSON() should return error for invalid JSON")
	}
}

func TestReservation_ToBooking(t *testing.T) {
	r := newValidReservation()
	b := r.ToBooking()

	if b.ID != r.BookingID {
		t.Errorf("ID = %v, want %v", b.ID, r.BookingID)
	}
	if b.UserID != r.UserID {
		t.Errorf("UserID = %v, want %v", b.UserID, r.UserID)
	}
	if b.EventID != r.EventID {
		t.Errorf("EventID = %v, want %v", b.EventID, r.EventID)
	}
	if b.ZoneID != r.ZoneID {
		t.Errorf("ZoneID = %v, want %v", b.ZoneID, r.ZoneID)
	}
	if b.Quantity != r.Quantity {
		t.Errorf("Quantity = %v, want %v", b.Quantity, r.Quantity)
	}
	if b.Status != BookingStatusReserved {
		t.Errorf("Status = %v, want %v", b.Status, BookingStatusReserved)
	}
	if b.TotalPrice != r.TotalPrice {
		t.Errorf("TotalPrice = %v, want %v", b.TotalPrice, r.TotalPrice)
	}
}
