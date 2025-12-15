package dto

import (
	"time"
)

// Topic names for payment events
const (
	TopicSeatRelease     = "payment.seat-release"
	TopicPaymentSuccess  = "payment.success"
)

// SeatReleaseReason represents the reason for releasing seats
type SeatReleaseReason string

const (
	SeatReleaseReasonPaymentFailed   SeatReleaseReason = "payment_failed"
	SeatReleaseReasonPaymentCanceled SeatReleaseReason = "payment_canceled"
	SeatReleaseReasonPaymentRefunded SeatReleaseReason = "payment_refunded"
)

// SeatReleaseEvent is published when seats need to be released due to payment failure
type SeatReleaseEvent struct {
	EventType   string            `json:"event_type"`
	BookingID   string            `json:"booking_id"`
	PaymentID   string            `json:"payment_id"`
	UserID      string            `json:"user_id,omitempty"`
	Reason      SeatReleaseReason `json:"reason"`
	FailureCode string            `json:"failure_code,omitempty"`
	Message     string            `json:"message,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Key returns the Kafka message key for partitioning
func (e *SeatReleaseEvent) Key() string {
	return e.BookingID
}

// PaymentSuccessEvent is published when payment succeeds to trigger post-payment saga
type PaymentSuccessEvent struct {
	EventType             string    `json:"event_type"`
	BookingID             string    `json:"booking_id"`
	PaymentID             string    `json:"payment_id"`
	StripePaymentIntentID string    `json:"stripe_payment_intent_id"`
	UserID                string    `json:"user_id,omitempty"`
	Amount                int64     `json:"amount"`
	Currency              string    `json:"currency"`
	Timestamp             time.Time `json:"timestamp"`
}

// Key returns the Kafka message key for partitioning
func (e *PaymentSuccessEvent) Key() string {
	return e.BookingID
}
