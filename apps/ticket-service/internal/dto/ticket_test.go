package dto

import (
	"testing"
	"time"
)

func TestCreateTicketTypeRequest_Validate(t *testing.T) {
	saleStart := time.Now().Add(24 * time.Hour)
	saleEnd := time.Now().Add(48 * time.Hour)

	tests := []struct {
		name    string
		req     CreateTicketTypeRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid request",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 100,
				MaxPerBooking: 4,
				SaleStartTime: saleStart,
				SaleEndTime:   saleEnd,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "missing event_id",
			req: CreateTicketTypeRequest{
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 100,
				MaxPerBooking: 4,
				SaleStartTime: saleStart,
				SaleEndTime:   saleEnd,
			},
			want:    false,
			wantMsg: "Event ID is required",
		},
		{
			name: "missing name",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Price:         100.00,
				TotalQuantity: 100,
				MaxPerBooking: 4,
				SaleStartTime: saleStart,
				SaleEndTime:   saleEnd,
			},
			want:    false,
			wantMsg: "Ticket type name is required",
		},
		{
			name: "negative price",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         -10.00,
				TotalQuantity: 100,
				MaxPerBooking: 4,
				SaleStartTime: saleStart,
				SaleEndTime:   saleEnd,
			},
			want:    false,
			wantMsg: "Price cannot be negative",
		},
		{
			name: "zero total quantity",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 0,
				MaxPerBooking: 4,
				SaleStartTime: saleStart,
				SaleEndTime:   saleEnd,
			},
			want:    false,
			wantMsg: "Total quantity must be at least 1",
		},
		{
			name: "zero max per booking",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 100,
				MaxPerBooking: 0,
				SaleStartTime: saleStart,
				SaleEndTime:   saleEnd,
			},
			want:    false,
			wantMsg: "Max per booking must be at least 1",
		},
		{
			name: "max per booking exceeds total",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 10,
				MaxPerBooking: 20,
				SaleStartTime: saleStart,
				SaleEndTime:   saleEnd,
			},
			want:    false,
			wantMsg: "Max per booking cannot exceed total quantity",
		},
		{
			name: "missing sale_start_time",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 100,
				MaxPerBooking: 4,
				SaleEndTime:   saleEnd,
			},
			want:    false,
			wantMsg: "Sale start time is required",
		},
		{
			name: "missing sale_end_time",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 100,
				MaxPerBooking: 4,
				SaleStartTime: saleStart,
			},
			want:    false,
			wantMsg: "Sale end time is required",
		},
		{
			name: "sale_end_time before sale_start_time",
			req: CreateTicketTypeRequest{
				EventID:       "event-123",
				Name:          "VIP",
				Price:         100.00,
				TotalQuantity: 100,
				MaxPerBooking: 4,
				SaleStartTime: saleEnd,
				SaleEndTime:   saleStart,
			},
			want:    false,
			wantMsg: "Sale end time must be after sale start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() got = %v, want %v", got, tt.want)
			}
			if msg != tt.wantMsg {
				t.Errorf("Validate() msg = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}

func TestUpdateTicketTypeRequest_Validate(t *testing.T) {
	price := 150.00
	totalQty := 200
	maxPerBooking := 6
	negativePrice := -10.00
	zeroQty := 0

	tests := []struct {
		name    string
		req     UpdateTicketTypeRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid name update",
			req: UpdateTicketTypeRequest{
				Name: "VIP Gold",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid price update",
			req: UpdateTicketTypeRequest{
				Price: &price,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid quantity update",
			req: UpdateTicketTypeRequest{
				TotalQuantity: &totalQty,
				MaxPerBooking: &maxPerBooking,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name:    "empty request",
			req:     UpdateTicketTypeRequest{},
			want:    false,
			wantMsg: "At least one field must be provided for update",
		},
		{
			name: "negative price",
			req: UpdateTicketTypeRequest{
				Price: &negativePrice,
			},
			want:    false,
			wantMsg: "Price cannot be negative",
		},
		{
			name: "zero total quantity",
			req: UpdateTicketTypeRequest{
				TotalQuantity: &zeroQty,
			},
			want:    false,
			wantMsg: "Total quantity must be at least 1",
		},
		{
			name: "zero max per booking",
			req: UpdateTicketTypeRequest{
				MaxPerBooking: &zeroQty,
			},
			want:    false,
			wantMsg: "Max per booking must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() got = %v, want %v", got, tt.want)
			}
			if msg != tt.wantMsg {
				t.Errorf("Validate() msg = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}
