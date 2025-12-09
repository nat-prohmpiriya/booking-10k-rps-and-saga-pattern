package dto

import (
	"testing"
	"time"
)

func TestCreateEventRequest_Validate(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)
	futureEndTime := time.Now().Add(48 * time.Hour)
	pastTime := time.Now().Add(-24 * time.Hour)

	tests := []struct {
		name    string
		req     CreateEventRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid request",
			req: CreateEventRequest{
				Name:      "Concert",
				VenueID:   "venue-123",
				StartTime: futureTime,
				EndTime:   futureEndTime,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "missing name",
			req: CreateEventRequest{
				VenueID:   "venue-123",
				StartTime: futureTime,
				EndTime:   futureEndTime,
			},
			want:    false,
			wantMsg: "Event name is required",
		},
		{
			name: "missing venue_id",
			req: CreateEventRequest{
				Name:      "Concert",
				StartTime: futureTime,
				EndTime:   futureEndTime,
			},
			want:    false,
			wantMsg: "Venue ID is required",
		},
		{
			name: "missing start_time",
			req: CreateEventRequest{
				Name:    "Concert",
				VenueID: "venue-123",
				EndTime: futureEndTime,
			},
			want:    false,
			wantMsg: "Start time is required",
		},
		{
			name: "missing end_time",
			req: CreateEventRequest{
				Name:      "Concert",
				VenueID:   "venue-123",
				StartTime: futureTime,
			},
			want:    false,
			wantMsg: "End time is required",
		},
		{
			name: "end_time before start_time",
			req: CreateEventRequest{
				Name:      "Concert",
				VenueID:   "venue-123",
				StartTime: futureEndTime,
				EndTime:   futureTime,
			},
			want:    false,
			wantMsg: "End time must be after start time",
		},
		{
			name: "start_time in the past",
			req: CreateEventRequest{
				Name:      "Concert",
				VenueID:   "venue-123",
				StartTime: pastTime,
				EndTime:   futureTime,
			},
			want:    false,
			wantMsg: "Start time must be in the future",
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

func TestUpdateEventRequest_Validate(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)
	futureEndTime := time.Now().Add(48 * time.Hour)

	tests := []struct {
		name    string
		req     UpdateEventRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid name update",
			req: UpdateEventRequest{
				Name: "Updated Concert",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid description update",
			req: UpdateEventRequest{
				Description: "New description",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid time update",
			req: UpdateEventRequest{
				StartTime: futureTime,
				EndTime:   futureEndTime,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name:    "empty request",
			req:     UpdateEventRequest{},
			want:    false,
			wantMsg: "At least one field must be provided for update",
		},
		{
			name: "end_time before start_time",
			req: UpdateEventRequest{
				StartTime: futureEndTime,
				EndTime:   futureTime,
			},
			want:    false,
			wantMsg: "End time must be after start time",
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
