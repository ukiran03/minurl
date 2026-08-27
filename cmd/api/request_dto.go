package main

import (
	"time"

	"ukiran.com/minurl/internal/validator"
)

// The Request Data Transfer Object (DTO)
type requestDTO struct {
	URL    string  `json:"url"`
	Expiry *string `json:"expires_at"`
}

func (req *requestDTO) Validate(v *validator.Validator) {
	// URL validation
	v.Check(req.URL != "", "url", "must be provided")
	v.Check(
		len(req.URL) >= 11 && len(req.URL) <= 2048,
		"url", "must be between 11 and 2048 characters long",
	)
}

// The Get Request DTO
type GetDTO struct {
	Slug  string
	From  time.Time
	To    time.Time
	Limit int
}

const maxLimit = 65536

// Validate checks business rules for the query parameters.
func (gd *GetDTO) Validate(v *validator.Validator) {
	// Ensure limit doesn't exceed the max threshold
	v.Check(gd.Limit <= maxLimit, "limit", "limit cannot exceed 65536")

	// Ensure limit isn't negative or zero (if a positive limit is required)
	v.Check(gd.Limit > 0, "limit", "limit must be greater than 0")

	// Ensure 'from' is before 'to' if both are provided
	if !gd.From.IsZero() && !gd.To.IsZero() {
		v.Check(
			gd.From.Before(gd.To),
			"from",
			"'from' time must be before 'to' time",
		)
	}
}
