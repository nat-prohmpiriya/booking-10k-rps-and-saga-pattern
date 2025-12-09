package domain

import (
	"testing"
	"time"
)

func TestTicketType_AvailableQuantity(t *testing.T) {
	tests := []struct {
		name          string
		totalQuantity int
		soldQuantity  int
		want          int
	}{
		{
			name:          "all available",
			totalQuantity: 100,
			soldQuantity:  0,
			want:          100,
		},
		{
			name:          "some sold",
			totalQuantity: 100,
			soldQuantity:  30,
			want:          70,
		},
		{
			name:          "all sold",
			totalQuantity: 100,
			soldQuantity:  100,
			want:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketType := &TicketType{
				TotalQuantity: tt.totalQuantity,
				SoldQuantity:  tt.soldQuantity,
			}
			if got := ticketType.AvailableQuantity(); got != tt.want {
				t.Errorf("AvailableQuantity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTicketType_IsAvailable(t *testing.T) {
	now := time.Now()
	pastStart := now.Add(-24 * time.Hour)
	futureEnd := now.Add(24 * time.Hour)
	pastEnd := now.Add(-1 * time.Hour)
	futureStart := now.Add(1 * time.Hour)

	tests := []struct {
		name   string
		ticket TicketType
		want   bool
	}{
		{
			name: "available",
			ticket: TicketType{
				Status:        TicketTypeStatusActive,
				TotalQuantity: 100,
				SoldQuantity:  50,
				SaleStartTime: pastStart,
				SaleEndTime:   futureEnd,
			},
			want: true,
		},
		{
			name: "inactive status",
			ticket: TicketType{
				Status:        TicketTypeStatusInactive,
				TotalQuantity: 100,
				SoldQuantity:  50,
				SaleStartTime: pastStart,
				SaleEndTime:   futureEnd,
			},
			want: false,
		},
		{
			name: "sold out",
			ticket: TicketType{
				Status:        TicketTypeStatusActive,
				TotalQuantity: 100,
				SoldQuantity:  100,
				SaleStartTime: pastStart,
				SaleEndTime:   futureEnd,
			},
			want: false,
		},
		{
			name: "sale not started",
			ticket: TicketType{
				Status:        TicketTypeStatusActive,
				TotalQuantity: 100,
				SoldQuantity:  50,
				SaleStartTime: futureStart,
				SaleEndTime:   futureEnd,
			},
			want: false,
		},
		{
			name: "sale ended",
			ticket: TicketType{
				Status:        TicketTypeStatusActive,
				TotalQuantity: 100,
				SoldQuantity:  50,
				SaleStartTime: pastStart,
				SaleEndTime:   pastEnd,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ticket.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
