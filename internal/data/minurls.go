package data

type MinUrl struct {
	Slug     string  `json:"slug"`
	URL      string  `json:"url"`
	Title    *string `json:"title,omitzero"`
	IsCustom bool    `json:"is_custom"`
	UserID   *int64  `json:"user_id,omitzero"`
	Lifespan
}
