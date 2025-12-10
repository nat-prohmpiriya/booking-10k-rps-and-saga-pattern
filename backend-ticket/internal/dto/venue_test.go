package dto

import "testing"

func TestCreateVenueRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateVenueRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid request",
			req: CreateVenueRequest{
				Name:     "Main Stadium",
				Address:  "123 Stadium Road",
				Capacity: 50000,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "missing name",
			req: CreateVenueRequest{
				Address:  "123 Stadium Road",
				Capacity: 50000,
			},
			want:    false,
			wantMsg: "Venue name is required",
		},
		{
			name: "missing address",
			req: CreateVenueRequest{
				Name:     "Main Stadium",
				Capacity: 50000,
			},
			want:    false,
			wantMsg: "Address is required",
		},
		{
			name: "zero capacity",
			req: CreateVenueRequest{
				Name:     "Main Stadium",
				Address:  "123 Stadium Road",
				Capacity: 0,
			},
			want:    false,
			wantMsg: "Capacity must be at least 1",
		},
		{
			name: "negative capacity",
			req: CreateVenueRequest{
				Name:     "Main Stadium",
				Address:  "123 Stadium Road",
				Capacity: -10,
			},
			want:    false,
			wantMsg: "Capacity must be at least 1",
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

func TestUpdateVenueRequest_Validate(t *testing.T) {
	validCapacity := 60000
	zeroCapacity := 0

	tests := []struct {
		name    string
		req     UpdateVenueRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid name update",
			req: UpdateVenueRequest{
				Name: "New Stadium Name",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid address update",
			req: UpdateVenueRequest{
				Address: "456 New Road",
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "valid capacity update",
			req: UpdateVenueRequest{
				Capacity: &validCapacity,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name:    "empty request",
			req:     UpdateVenueRequest{},
			want:    false,
			wantMsg: "At least one field must be provided for update",
		},
		{
			name: "zero capacity",
			req: UpdateVenueRequest{
				Capacity: &zeroCapacity,
			},
			want:    false,
			wantMsg: "Capacity must be at least 1",
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

func TestCreateZoneRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateZoneRequest
		want    bool
		wantMsg string
	}{
		{
			name: "valid request",
			req: CreateZoneRequest{
				VenueID:  "venue-123",
				Name:     "Zone A",
				Capacity: 1000,
			},
			want:    true,
			wantMsg: "",
		},
		{
			name: "missing venue_id",
			req: CreateZoneRequest{
				Name:     "Zone A",
				Capacity: 1000,
			},
			want:    false,
			wantMsg: "Venue ID is required",
		},
		{
			name: "missing name",
			req: CreateZoneRequest{
				VenueID:  "venue-123",
				Capacity: 1000,
			},
			want:    false,
			wantMsg: "Zone name is required",
		},
		{
			name: "zero capacity",
			req: CreateZoneRequest{
				VenueID:  "venue-123",
				Name:     "Zone A",
				Capacity: 0,
			},
			want:    false,
			wantMsg: "Capacity must be at least 1",
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
