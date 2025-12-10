package domain

import (
	"time"
)

// BookingEventType represents the type of booking event
type BookingEventType string

const (
	BookingEventCreated   BookingEventType = "booking.created"
	BookingEventConfirmed BookingEventType = "booking.confirmed"
	BookingEventCancelled BookingEventType = "booking.cancelled"
	BookingEventExpired   BookingEventType = "booking.expired"
)

// BookingEvent represents a booking domain event
type BookingEvent struct {
	EventID     string           `json:"event_id"`
	EventType   BookingEventType `json:"event_type"`
	OccurredAt  time.Time        `json:"occurred_at"`
	Version     int              `json:"version"`
	BookingData *BookingEventData `json:"data"`
}

// BookingEventData contains the booking data in the event
type BookingEventData struct {
	BookingID        string    `json:"booking_id"`
	TenantID         string    `json:"tenant_id,omitempty"`
	UserID           string    `json:"user_id"`
	EventID          string    `json:"event_id"`
	ShowID           string    `json:"show_id,omitempty"`
	ZoneID           string    `json:"zone_id"`
	Quantity         int       `json:"quantity"`
	UnitPrice        float64   `json:"unit_price"`
	TotalPrice       float64   `json:"total_price"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	PaymentID        string    `json:"payment_id,omitempty"`
	ConfirmationCode string    `json:"confirmation_code,omitempty"`
	ReservedAt       time.Time `json:"reserved_at"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// NewBookingEvent creates a new booking event from a booking
func NewBookingEvent(eventType BookingEventType, booking *Booking, eventID string) *BookingEvent {
	return &BookingEvent{
		EventID:    eventID,
		EventType:  eventType,
		OccurredAt: time.Now(),
		Version:    1,
		BookingData: &BookingEventData{
			BookingID:        booking.ID,
			TenantID:         booking.TenantID,
			UserID:           booking.UserID,
			EventID:          booking.EventID,
			ShowID:           booking.ShowID,
			ZoneID:           booking.ZoneID,
			Quantity:         booking.Quantity,
			UnitPrice:        booking.UnitPrice,
			TotalPrice:       booking.TotalPrice,
			Currency:         booking.Currency,
			Status:           string(booking.Status),
			PaymentID:        booking.PaymentID,
			ConfirmationCode: booking.ConfirmationCode,
			ReservedAt:       booking.ReservedAt,
			ConfirmedAt:      booking.ConfirmedAt,
			CancelledAt:      booking.CancelledAt,
			ExpiresAt:        booking.ExpiresAt,
		},
	}
}

// Topic returns the Kafka topic for this event type
func (e *BookingEvent) Topic() string {
	return "booking-events"
}

// Key returns the partition key for this event (booking ID)
func (e *BookingEvent) Key() string {
	if e.BookingData != nil {
		return e.BookingData.BookingID
	}
	return e.EventID
}
