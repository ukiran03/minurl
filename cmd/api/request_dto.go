package main

import (
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
