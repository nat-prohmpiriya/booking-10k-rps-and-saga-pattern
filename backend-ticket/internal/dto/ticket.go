package dto

import "time"

// CreateTicketTypeRequest represents the request to create a new ticket type
type CreateTicketTypeRequest struct {
	EventID       string    `json:"event_id" binding:"required"`
	ZoneID        string    `json:"zone_id"`
	Name          string    `json:"name" binding:"required,min=1,max=100"`
	Description   string    `json:"description" binding:"max=500"`
	Price         float64   `json:"price" binding:"required,gte=0"`
	TotalQuantity int       `json:"total_quantity" binding:"required,gte=1"`
	MaxPerBooking int       `json:"max_per_booking" binding:"required,gte=1"`
	SaleStartTime time.Time `json:"sale_start_time" binding:"required"`
	SaleEndTime   time.Time `json:"sale_end_time" binding:"required"`
}

// Validate validates the CreateTicketTypeRequest
func (r *CreateTicketTypeRequest) Validate() (bool, string) {
	if r.EventID == "" {
		return false, "Event ID is required"
	}
	if r.Name == "" {
		return false, "Ticket type name is required"
	}
	if r.Price < 0 {
		return false, "Price cannot be negative"
	}
	if r.TotalQuantity < 1 {
		return false, "Total quantity must be at least 1"
	}
	if r.MaxPerBooking < 1 {
		return false, "Max per booking must be at least 1"
	}
	if r.MaxPerBooking > r.TotalQuantity {
		return false, "Max per booking cannot exceed total quantity"
	}
	if r.SaleStartTime.IsZero() {
		return false, "Sale start time is required"
	}
	if r.SaleEndTime.IsZero() {
		return false, "Sale end time is required"
	}
	if r.SaleEndTime.Before(r.SaleStartTime) {
		return false, "Sale end time must be after sale start time"
	}
	return true, ""
}

// UpdateTicketTypeRequest represents the request to update a ticket type
type UpdateTicketTypeRequest struct {
	Name          string    `json:"name" binding:"omitempty,min=1,max=100"`
	Description   string    `json:"description" binding:"max=500"`
	Price         *float64  `json:"price" binding:"omitempty,gte=0"`
	TotalQuantity *int      `json:"total_quantity" binding:"omitempty,gte=1"`
	MaxPerBooking *int      `json:"max_per_booking" binding:"omitempty,gte=1"`
	SaleStartTime time.Time `json:"sale_start_time"`
	SaleEndTime   time.Time `json:"sale_end_time"`
	Status        string    `json:"status"`
}

// Validate validates the UpdateTicketTypeRequest
func (r *UpdateTicketTypeRequest) Validate() (bool, string) {
	hasUpdate := r.Name != "" ||
		r.Description != "" ||
		r.Price != nil ||
		r.TotalQuantity != nil ||
		r.MaxPerBooking != nil ||
		!r.SaleStartTime.IsZero() ||
		!r.SaleEndTime.IsZero() ||
		r.Status != ""

	if !hasUpdate {
		return false, "At least one field must be provided for update"
	}

	if r.Price != nil && *r.Price < 0 {
		return false, "Price cannot be negative"
	}
	if r.TotalQuantity != nil && *r.TotalQuantity < 1 {
		return false, "Total quantity must be at least 1"
	}
	if r.MaxPerBooking != nil && *r.MaxPerBooking < 1 {
		return false, "Max per booking must be at least 1"
	}
	if !r.SaleStartTime.IsZero() && !r.SaleEndTime.IsZero() && r.SaleEndTime.Before(r.SaleStartTime) {
		return false, "Sale end time must be after sale start time"
	}
	return true, ""
}

// TicketTypeResponse represents the response for a ticket type
type TicketTypeResponse struct {
	ID                string  `json:"id"`
	EventID           string  `json:"event_id"`
	ZoneID            string  `json:"zone_id,omitempty"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	Price             float64 `json:"price"`
	TotalQuantity     int     `json:"total_quantity"`
	SoldQuantity      int     `json:"sold_quantity"`
	AvailableQuantity int     `json:"available_quantity"`
	MaxPerBooking     int     `json:"max_per_booking"`
	SaleStartTime     string  `json:"sale_start_time"`
	SaleEndTime       string  `json:"sale_end_time"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

// AvailabilityRequest represents the request to check ticket availability
type AvailabilityRequest struct {
	EventID      string `json:"event_id" binding:"required"`
	TicketTypeID string `json:"ticket_type_id" binding:"required"`
	Quantity     int    `json:"quantity" binding:"required,gte=1"`
}

// AvailabilityResponse represents the response for availability check
type AvailabilityResponse struct {
	Available         bool    `json:"available"`
	AvailableQuantity int     `json:"available_quantity"`
	RequestedQuantity int     `json:"requested_quantity"`
	PricePerTicket    float64 `json:"price_per_ticket"`
	TotalPrice        float64 `json:"total_price"`
	Message           string  `json:"message,omitempty"`
}
