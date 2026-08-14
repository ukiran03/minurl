package data

import "ukiran.com/minurl/internal/flake"

type MinUrl struct {
	Flake   int64    `json:"flake"    redis:"flake"`
	Slug    string   `json:"slug"     redis:"slug"`
	URL     string   `json:"url"      redis:"url"`
	URLHash string   `json:"url_hash" redis:"url_hash"`
	Life    Lifespan `json:"lifespan" redis:"life"`
}

func NewMinUrl(
	sfnid int64, longURL string, lifespan Lifespan,
) *MinUrl {
	flake, _ := flake.NewFlake(int64(sfnid))
	slug := flake.Base62()

	return &MinUrl{
		Flake: int64(flake),
		Slug:  slug,
		URL:   longURL,
		Life:  lifespan,
	}
}
