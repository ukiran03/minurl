package main

import (
	"ukiran.com/minurl/internal/validator"
)

// The Request Data Transfer Object (DTO)
type minurlRequestDTO struct {
	Slug     string  `json:"slug,omitzero"`
	URL      string  `json:"url"`
	Title    *string `json:"title,omitzero"`
	IsCustom bool    `json:"is_custom"`
	Expiry   *string `json:"expires_at"`
	UserID   *int64  `json:"user_id,omitzero"`
}

func (req *minurlRequestDTO) Validate(v *validator.Validator) {
	// URL validation
	v.Check(req.URL != "", "url", "must be provided")
	v.Check(
		len(req.URL) >= 11 && len(req.URL) <= 2048,
		"url", "must be between 11 and 2048 characters long",
	)

	// Slug validation
	if req.IsCustom {
		v.Check(len(req.Slug) >= 8 && len(req.Slug) <= 100,
			"custom slug", "must be between 8 and 100 characters long")
		v.Check(validator.Matches(req.Slug, validator.SlugRegex),
			"custom slug", ErrInvalidCustomSlug.Error())
	}

	// Title validation (Using pointer safe-check)
	if req.Title != nil && *req.Title != "" {
		v.Check(len(*req.Title) <= 100,
			"title", "must not be more than 100 characters long")
	}

	// UserID validation
	if req.UserID != nil {
		v.Check(*req.UserID > 0, "user_id", "must be a valid positive integer")
	}
}
