package domain

import (
	"encoding/json"
	"strings"
	"time"
)

// Reservation represents a temporary seat reservation in Redis
// This is stored in Redis with TTL and used before confirmation
type Reservation struct {
	BookingID  string    `json:"booking_id"`
	UserID     string    `json:"user_id"`
	EventID    string    `json:"event_id"`
	ZoneID     string    `json:"zone_id"`
	Quantity   int       `json:"quantity"`
	UnitPrice  float64   `json:"unit_price"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"` // "reserved", "confirmed", "released"
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Reservation status constants
const (
	ReservationStatusReserved  = "reserved"
	ReservationStatusConfirmed = "confirmed"
	ReservationStatusReleased  = "released"
)

// NewReservation creates a new Reservation with the given parameters
func NewReservation(bookingID, userID, eventID, zoneID string, quantity int, unitPrice float64, ttl time.Duration) *Reservation {
	now := time.Now()
	return &Reservation{
		BookingID:  bookingID,
		UserID:     userID,
		EventID:    eventID,
		ZoneID:     zoneID,
		Quantity:   quantity,
		UnitPrice:  unitPrice,
		TotalPrice: unitPrice * float64(quantity),
		Status:     ReservationStatusReserved,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
	}
}

// Validate validates all reservation fields
func (r *Reservation) Validate() error {
	if err := r.ValidateBookingID(); err != nil {
		return err
	}
	if err := r.ValidateUserID(); err != nil {
		return err
	}
	if err := r.ValidateEventID(); err != nil {
		return err
	}
	if err := r.ValidateZoneID(); err != nil {
		return err
	}
	if err := r.ValidateQuantity(); err != nil {
		return err
	}
	if err := r.ValidateUnitPrice(); err != nil {
		return err
	}
	return nil
}

// ValidateBookingID validates the booking ID
func (r *Reservation) ValidateBookingID() error {
	if strings.TrimSpace(r.BookingID) == "" {
		return ErrInvalidBookingID
	}
	return nil
}

// ValidateUserID validates the user ID
func (r *Reservation) ValidateUserID() error {
	if strings.TrimSpace(r.UserID) == "" {
		return ErrInvalidUserID
	}
	return nil
}

// ValidateEventID validates the event ID
func (r *Reservation) ValidateEventID() error {
	if strings.TrimSpace(r.EventID) == "" {
		return ErrInvalidEventID
	}
	return nil
}

// ValidateZoneID validates the zone ID
func (r *Reservation) ValidateZoneID() error {
	if strings.TrimSpace(r.ZoneID) == "" {
		return ErrInvalidZoneID
	}
	return nil
}

// ValidateQuantity validates the reservation quantity
func (r *Reservation) ValidateQuantity() error {
	if r.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	return nil
}

// ValidateUnitPrice validates the unit price
func (r *Reservation) ValidateUnitPrice() error {
	if r.UnitPrice < 0 {
		return ErrInvalidUnitPrice
	}
	return nil
}

// IsExpired checks if the reservation has expired
func (r *Reservation) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsExpiredAt checks if the reservation is expired at a specific time
func (r *Reservation) IsExpiredAt(t time.Time) bool {
	return t.After(r.ExpiresAt)
}

// IsReserved checks if the reservation is in reserved status
func (r *Reservation) IsReserved() bool {
	return r.Status == ReservationStatusReserved
}

// IsConfirmed checks if the reservation is in confirmed status
func (r *Reservation) IsConfirmed() bool {
	return r.Status == ReservationStatusConfirmed
}

// IsReleased checks if the reservation is in released status
func (r *Reservation) IsReleased() bool {
	return r.Status == ReservationStatusReleased
}

// CanConfirm checks if the reservation can be confirmed
func (r *Reservation) CanConfirm() bool {
	return r.Status == ReservationStatusReserved && !r.IsExpired()
}

// CanRelease checks if the reservation can be released
func (r *Reservation) CanRelease() bool {
	return r.Status == ReservationStatusReserved
}

// TimeUntilExpiry returns the duration until the reservation expires
func (r *Reservation) TimeUntilExpiry() time.Duration {
	return time.Until(r.ExpiresAt)
}

// TTL returns the remaining TTL in seconds
func (r *Reservation) TTL() int64 {
	ttl := r.TimeUntilExpiry()
	if ttl < 0 {
		return 0
	}
	return int64(ttl.Seconds())
}

// BelongsToUser checks if the reservation belongs to the specified user
func (r *Reservation) BelongsToUser(userID string) bool {
	return r.UserID == userID
}

// ToJSON serializes the reservation to JSON
func (r *Reservation) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReservationFromJSON deserializes a reservation from JSON
func ReservationFromJSON(data string) (*Reservation, error) {
	var r Reservation
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ToBooking converts a Reservation to a Booking entity
func (r *Reservation) ToBooking() *Booking {
	return &Booking{
		ID:         r.BookingID,
		UserID:     r.UserID,
		EventID:    r.EventID,
		ZoneID:     r.ZoneID,
		Quantity:   r.Quantity,
		Status:     BookingStatusReserved,
		TotalPrice: r.TotalPrice,
		ReservedAt: r.CreatedAt,
		ExpiresAt:  r.ExpiresAt,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.CreatedAt,
	}
}
