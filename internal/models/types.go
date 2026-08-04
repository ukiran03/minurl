package models

import "ukiran.com/minurl/internal/flake"

type MinUrl struct {
	Flake   int64    `json:"flake"`
	Slug    string   `json:"slug"`
	URL     string   `json:"url"`
	URLHash string   `json:"url_hash"`
	Life    Lifespan `json:"lifespan"`
}

func NewMinUrl(
	sfnid int64, longURL string, lifespan Lifespan,
) *MinUrl {
	flake := flake.NewFlake(int64(sfnid))
	slug := flake.Base62()

	return &MinUrl{
		Flake: int64(flake),
		Slug:  slug,
		URL:   longURL,
		Life:  lifespan,
	}
}
