package main

import (
	"ukiran.com/minurl/internal/validator"
)

// The Request Data Transfer Object (DTO)
type minurlRequestDTO struct {
	Slug   *string `json:"slug,omitzero"`
	URL    string  `json:"url"`
	Title  *string `json:"title,omitzero"`
	Expiry *string `json:"expires_at"`
	UserID *int64  `json:"user_id,omitzero"`
}

func (req *minurlRequestDTO) Validate(v *validator.Validator) {
	// URL validation
	v.Check(req.URL != "", "url", "must be provided")
	v.Check(
		len(req.URL) >= 11 && len(req.URL) <= 2048,
		"url", "must be between 11 and 2048 characters long",
	)

	isCustom := (req.Slug != nil) && (*req.Slug != "")

	// Custom slug validation (if any)
	if isCustom {
		slugVal := *req.Slug
		slugLen := len(slugVal)
		v.Check(slugLen >= 8 && slugLen <= 100,
			"custom slug", "must be between 8 and 100 characters long")
		v.Check(validator.Matches(slugVal, validator.SlugRegex),
			"custom slug", ErrInvalidCustomSlug.Error())
	}

	// Title validation (Using pointer safe-check)
	if req.Title != nil && *req.Title != "" {
		v.Check(len(*req.Title) <= 100,
			"title", "must not be more than 100 characters long")
	}

	// UserID validation
	if isCustom && req.UserID == nil {
		v.AddError("user_id", "custom_slug provider must be a valid user")
	} else if req.UserID != nil {
		// Only checks if req.UserID actually holds a pointer
		v.Check(*req.UserID > 0, "user_id", "must be a valid positive integer")
	}
}
