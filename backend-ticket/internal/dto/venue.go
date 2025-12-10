package dto

// CreateVenueRequest represents the request to create a new venue
type CreateVenueRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=200"`
	Address  string `json:"address" binding:"required,max=500"`
	Capacity int    `json:"capacity" binding:"required,gte=1"`
	TenantID string `json:"-"` // Set from context
}

// Validate validates the CreateVenueRequest
func (r *CreateVenueRequest) Validate() (bool, string) {
	if r.Name == "" {
		return false, "Venue name is required"
	}
	if r.Address == "" {
		return false, "Address is required"
	}
	if r.Capacity < 1 {
		return false, "Capacity must be at least 1"
	}
	return true, ""
}

// UpdateVenueRequest represents the request to update a venue
type UpdateVenueRequest struct {
	Name     string `json:"name" binding:"omitempty,min=1,max=200"`
	Address  string `json:"address" binding:"max=500"`
	Capacity *int   `json:"capacity" binding:"omitempty,gte=1"`
}

// Validate validates the UpdateVenueRequest
func (r *UpdateVenueRequest) Validate() (bool, string) {
	if r.Name == "" && r.Address == "" && r.Capacity == nil {
		return false, "At least one field must be provided for update"
	}
	if r.Capacity != nil && *r.Capacity < 1 {
		return false, "Capacity must be at least 1"
	}
	return true, ""
}

// VenueResponse represents the response for a venue
type VenueResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Capacity  int    `json:"capacity"`
	TenantID  string `json:"tenant_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateZoneRequest represents the request to create a new zone
type CreateZoneRequest struct {
	VenueID  string `json:"venue_id" binding:"required"`
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Capacity int    `json:"capacity" binding:"required,gte=1"`
}

// Validate validates the CreateZoneRequest
func (r *CreateZoneRequest) Validate() (bool, string) {
	if r.VenueID == "" {
		return false, "Venue ID is required"
	}
	if r.Name == "" {
		return false, "Zone name is required"
	}
	if r.Capacity < 1 {
		return false, "Capacity must be at least 1"
	}
	return true, ""
}

// ZoneResponse represents the response for a zone
type ZoneResponse struct {
	ID        string `json:"id"`
	VenueID   string `json:"venue_id"`
	Name      string `json:"name"`
	Capacity  int    `json:"capacity"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
