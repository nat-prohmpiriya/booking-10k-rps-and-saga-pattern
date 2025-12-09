package domain

import (
	"testing"
	"time"
)

func TestBookingStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status BookingStatus
		want   bool
	}{
		{"reserved is valid", BookingStatusReserved, true},
		{"confirmed is valid", BookingStatusConfirmed, true},
		{"cancelled is valid", BookingStatusCancelled, true},
		{"expired is valid", BookingStatusExpired, true},
		{"empty is invalid", BookingStatus(""), false},
		{"random is invalid", BookingStatus("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("BookingStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBookingStatus_String(t *testing.T) {
	tests := []struct {
		status BookingStatus
		want   string
	}{
		{BookingStatusReserved, "reserved"},
		{BookingStatusConfirmed, "confirmed"},
		{BookingStatusCancelled, "cancelled"},
		{BookingStatusExpired, "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("BookingStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newValidBooking() *Booking {
	return &Booking{
		ID:         "booking-123",
		UserID:     "user-456",
		EventID:    "event-789",
		ZoneID:     "zone-abc",
		Quantity:   2,
		Status:     BookingStatusReserved,
		TotalPrice: 100.00,
		ReservedAt: time.Now(),
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func TestBooking_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Booking)
		wantErr error
	}{
		{
			name:    "valid booking",
			modify:  func(b *Booking) {},
			wantErr: nil,
		},
		{
			name:    "empty ID",
			modify:  func(b *Booking) { b.ID = "" },
			wantErr: ErrInvalidBookingID,
		},
		{
			name:    "whitespace ID",
			modify:  func(b *Booking) { b.ID = "   " },
			wantErr: ErrInvalidBookingID,
		},
		{
			name:    "empty UserID",
			modify:  func(b *Booking) { b.UserID = "" },
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "empty EventID",
			modify:  func(b *Booking) { b.EventID = "" },
			wantErr: ErrInvalidEventID,
		},
		{
			name:    "empty ZoneID",
			modify:  func(b *Booking) { b.ZoneID = "" },
			wantErr: ErrInvalidZoneID,
		},
		{
			name:    "zero quantity",
			modify:  func(b *Booking) { b.Quantity = 0 },
			wantErr: ErrInvalidQuantity,
		},
		{
			name:    "negative quantity",
			modify:  func(b *Booking) { b.Quantity = -1 },
			wantErr: ErrInvalidQuantity,
		},
		{
			name:    "invalid status",
			modify:  func(b *Booking) { b.Status = "invalid" },
			wantErr: ErrInvalidBookingStatus,
		},
		{
			name:    "negative total price",
			modify:  func(b *Booking) { b.TotalPrice = -10.00 },
			wantErr: ErrInvalidTotalPrice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newValidBooking()
			tt.modify(b)
			err := b.Validate()
			if err != tt.wantErr {
				t.Errorf("Booking.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBooking_IsExpired(t *testing.T) {
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
			b := newValidBooking()
			b.ExpiresAt = tt.expiresAt
			if got := b.IsExpired(); got != tt.want {
				t.Errorf("Booking.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBooking_IsExpiredAt(t *testing.T) {
	b := newValidBooking()
	b.ExpiresAt = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

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
			if got := b.IsExpiredAt(tt.at); got != tt.want {
				t.Errorf("Booking.IsExpiredAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBooking_CanConfirm(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Booking)
		want   bool
	}{
		{
			name:   "reserved and not expired",
			modify: func(b *Booking) {},
			want:   true,
		},
		{
			name: "reserved but expired",
			modify: func(b *Booking) {
				b.ExpiresAt = time.Now().Add(-1 * time.Minute)
			},
			want: false,
		},
		{
			name:   "already confirmed",
			modify: func(b *Booking) { b.Status = BookingStatusConfirmed },
			want:   false,
		},
		{
			name:   "cancelled",
			modify: func(b *Booking) { b.Status = BookingStatusCancelled },
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newValidBooking()
			tt.modify(b)
			if got := b.CanConfirm(); got != tt.want {
				t.Errorf("Booking.CanConfirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBooking_CanCancel(t *testing.T) {
	tests := []struct {
		name   string
		status BookingStatus
		want   bool
	}{
		{"reserved", BookingStatusReserved, true},
		{"confirmed", BookingStatusConfirmed, false},
		{"cancelled", BookingStatusCancelled, false},
		{"expired", BookingStatusExpired, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newValidBooking()
			b.Status = tt.status
			if got := b.CanCancel(); got != tt.want {
				t.Errorf("Booking.CanCancel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBooking_StatusChecks(t *testing.T) {
	tests := []struct {
		status       BookingStatus
		isReserved   bool
		isConfirmed  bool
		isCancelled  bool
	}{
		{BookingStatusReserved, true, false, false},
		{BookingStatusConfirmed, false, true, false},
		{BookingStatusCancelled, false, false, true},
		{BookingStatusExpired, false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			b := newValidBooking()
			b.Status = tt.status

			if got := b.IsReserved(); got != tt.isReserved {
				t.Errorf("Booking.IsReserved() = %v, want %v", got, tt.isReserved)
			}
			if got := b.IsConfirmed(); got != tt.isConfirmed {
				t.Errorf("Booking.IsConfirmed() = %v, want %v", got, tt.isConfirmed)
			}
			if got := b.IsCancelled(); got != tt.isCancelled {
				t.Errorf("Booking.IsCancelled() = %v, want %v", got, tt.isCancelled)
			}
		})
	}
}

func TestBooking_Confirm(t *testing.T) {
	t.Run("successful confirmation", func(t *testing.T) {
		b := newValidBooking()
		paymentID := "payment-123"

		err := b.Confirm(paymentID)
		if err != nil {
			t.Errorf("Booking.Confirm() error = %v, want nil", err)
		}
		if b.Status != BookingStatusConfirmed {
			t.Errorf("status = %v, want %v", b.Status, BookingStatusConfirmed)
		}
		if b.PaymentID != paymentID {
			t.Errorf("paymentID = %v, want %v", b.PaymentID, paymentID)
		}
		if b.ConfirmedAt == nil {
			t.Error("confirmedAt should not be nil")
		}
	})

	t.Run("confirm expired booking", func(t *testing.T) {
		b := newValidBooking()
		b.ExpiresAt = time.Now().Add(-1 * time.Minute)

		err := b.Confirm("payment-123")
		if err != ErrBookingExpired {
			t.Errorf("Booking.Confirm() error = %v, want %v", err, ErrBookingExpired)
		}
	})

	t.Run("confirm already confirmed booking", func(t *testing.T) {
		b := newValidBooking()
		b.Status = BookingStatusConfirmed

		err := b.Confirm("payment-123")
		if err != ErrAlreadyConfirmed {
			t.Errorf("Booking.Confirm() error = %v, want %v", err, ErrAlreadyConfirmed)
		}
	})
}

func TestBooking_Cancel(t *testing.T) {
	t.Run("successful cancellation", func(t *testing.T) {
		b := newValidBooking()

		err := b.Cancel()
		if err != nil {
			t.Errorf("Booking.Cancel() error = %v, want nil", err)
		}
		if b.Status != BookingStatusCancelled {
			t.Errorf("status = %v, want %v", b.Status, BookingStatusCancelled)
		}
	})

	t.Run("cancel confirmed booking", func(t *testing.T) {
		b := newValidBooking()
		b.Status = BookingStatusConfirmed

		err := b.Cancel()
		if err != ErrAlreadyConfirmed {
			t.Errorf("Booking.Cancel() error = %v, want %v", err, ErrAlreadyConfirmed)
		}
	})

	t.Run("cancel already cancelled booking", func(t *testing.T) {
		b := newValidBooking()
		b.Status = BookingStatusCancelled

		err := b.Cancel()
		if err != ErrAlreadyReleased {
			t.Errorf("Booking.Cancel() error = %v, want %v", err, ErrAlreadyReleased)
		}
	})
}

func TestBooking_Expire(t *testing.T) {
	t.Run("successful expiration", func(t *testing.T) {
		b := newValidBooking()

		err := b.Expire()
		if err != nil {
			t.Errorf("Booking.Expire() error = %v, want nil", err)
		}
		if b.Status != BookingStatusExpired {
			t.Errorf("status = %v, want %v", b.Status, BookingStatusExpired)
		}
	})

	t.Run("expire confirmed booking", func(t *testing.T) {
		b := newValidBooking()
		b.Status = BookingStatusConfirmed

		err := b.Expire()
		if err != ErrInvalidBookingStatus {
			t.Errorf("Booking.Expire() error = %v, want %v", err, ErrInvalidBookingStatus)
		}
	})
}

func TestBooking_TimeUntilExpiry(t *testing.T) {
	b := newValidBooking()
	b.ExpiresAt = time.Now().Add(5 * time.Minute)

	ttl := b.TimeUntilExpiry()
	if ttl < 4*time.Minute || ttl > 5*time.Minute {
		t.Errorf("TimeUntilExpiry() = %v, expected around 5 minutes", ttl)
	}
}

func TestBooking_BelongsToUser(t *testing.T) {
	b := newValidBooking()
	b.UserID = "user-123"

	if !b.BelongsToUser("user-123") {
		t.Error("BelongsToUser() should return true for matching user")
	}
	if b.BelongsToUser("user-456") {
		t.Error("BelongsToUser() should return false for non-matching user")
	}
}
