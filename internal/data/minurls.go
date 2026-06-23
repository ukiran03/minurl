package data

import "time"

type MinUrl struct {
	Slug      string     `json:"slug"`
	URL       string     `json:"url"`
	Title     *string    `json:"title,omitzero"`
	IsCustom  bool       `json:"is_custom"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	UserID    *int64     `json:"user_id,omitzero"`
}
