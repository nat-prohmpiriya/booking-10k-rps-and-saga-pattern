package dto

import "time"

// CreateShowRequest represents the request to create a new show
type CreateShowRequest struct {
	EventID   string    `json:"-"` // Set from URL param
	Name      string    `json:"name" binding:"required,min=1,max=200"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
}

// Validate validates the CreateShowRequest
func (r *CreateShowRequest) Validate() (bool, string) {
	if r.Name == "" {
		return false, "Show name is required"
	}
	if r.StartTime.IsZero() {
		return false, "Start time is required"
	}
	if r.EndTime.IsZero() {
		return false, "End time is required"
	}
	if r.EndTime.Before(r.StartTime) {
		return false, "End time must be after start time"
	}
	return true, ""
}

// UpdateShowRequest represents the request to update a show
type UpdateShowRequest struct {
	Name      string    `json:"name" binding:"omitempty,min=1,max=200"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

// Validate validates the UpdateShowRequest
func (r *UpdateShowRequest) Validate() (bool, string) {
	if r.Name == "" && r.StartTime.IsZero() && r.EndTime.IsZero() && r.Status == "" {
		return false, "At least one field must be provided for update"
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() && r.EndTime.Before(r.StartTime) {
		return false, "End time must be after start time"
	}
	return true, ""
}

// ShowResponse represents the response for a show
type ShowResponse struct {
	ID        string `json:"id"`
	EventID   string `json:"event_id"`
	Name      string `json:"name"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ShowListResponse represents a list of shows
type ShowListResponse struct {
	Shows  []*ShowResponse `json:"shows"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// ShowListFilter represents filters for listing shows
type ShowListFilter struct {
	EventID string `form:"-"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}

// SetDefaults sets default values for pagination
func (f *ShowListFilter) SetDefaults() {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
