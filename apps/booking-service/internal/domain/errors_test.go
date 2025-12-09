package domain

import (
	"errors"
	"testing"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"booking not found", ErrBookingNotFound, true},
		{"reservation not found", ErrReservationNotFound, true},
		{"zone not found", ErrZoneNotFound, true},
		{"event not found", ErrEventNotFound, true},
		{"insufficient seats", ErrInsufficientSeats, false},
		{"invalid user id", ErrInvalidUserID, false},
		{"nil error", nil, false},
		{"random error", errors.New("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"invalid user id", ErrInvalidUserID, true},
		{"invalid booking id", ErrInvalidBookingID, true},
		{"invalid event id", ErrInvalidEventID, true},
		{"invalid zone id", ErrInvalidZoneID, true},
		{"invalid quantity", ErrInvalidQuantity, true},
		{"invalid total price", ErrInvalidTotalPrice, true},
		{"invalid unit price", ErrInvalidUnitPrice, true},
		{"invalid booking status", ErrInvalidBookingStatus, true},
		{"booking not found", ErrBookingNotFound, false},
		{"insufficient seats", ErrInsufficientSeats, false},
		{"nil error", nil, false},
		{"random error", errors.New("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidationError(tt.err); got != tt.want {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"already confirmed", ErrAlreadyConfirmed, true},
		{"already released", ErrAlreadyReleased, true},
		{"booking already exists", ErrBookingAlreadyExists, true},
		{"insufficient seats", ErrInsufficientSeats, true},
		{"max tickets exceeded", ErrMaxTicketsExceeded, true},
		{"booking not found", ErrBookingNotFound, false},
		{"invalid user id", ErrInvalidUserID, false},
		{"nil error", nil, false},
		{"random error", errors.New("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConflictError(tt.err); got != tt.want {
				t.Errorf("IsConflictError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExpiredError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"booking expired", ErrBookingExpired, true},
		{"reservation expired", ErrReservationExpired, true},
		{"booking not found", ErrBookingNotFound, false},
		{"insufficient seats", ErrInsufficientSeats, false},
		{"nil error", nil, false},
		{"random error", errors.New("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExpiredError(tt.err); got != tt.want {
				t.Errorf("IsExpiredError() = %v, want %v", got, tt.want)
			}
		})
	}
}
