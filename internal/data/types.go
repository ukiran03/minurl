package data

import (
	"net/http"
	"time"

	"ukiran.com/minurl/internal/flake"
)

// BatchItem defines the constraint for types allowed in a batch.
type BatchItem interface {
	MinUrl | ClickEvent
}

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

// ClickEvent is an append-only analytics record for ClickHouse.
type ClickEvent struct {
	Slug       string    `json:"slug"        ch:"slug"`
	Timestamp  time.Time `json:"timestamp"   ch:"timestamp"`
	RemoteAddr string    `json:"remote_addr" ch:"remote_addr"`
	UserAgent  string    `json:"user_agent"  ch:"user_agent"`
	Referrer   string    `json:"referrer"    ch:"referrer"`
}

func NewClickEvent(slug string, r *http.Request) *ClickEvent {
	remoteAddr := extractIP(r) // IP
	referrer := sanitizeReferrer(r.Referer())

	return &ClickEvent{
		Slug:       slug,
		Timestamp:  time.Now().UTC(),
		RemoteAddr: remoteAddr,
		UserAgent:  r.UserAgent(),
		Referrer:   referrer,
	}
}

// ClickStats is a read-model summary for a slug over a time window.
type ClickStats struct {
	Slug         string          `json:"slug"          ch:"slug"`
	From         time.Time       `json:"from"          ch:"from"`
	To           time.Time       `json:"to"            ch:"to"`
	TotalClicks  uint64          `json:"total_clicks"  ch:"total_clicks"`
	TopReferrers []ReferrerCount `json:"top_referrers" ch:"top_referrers"`
}

// ReferrerCount ranks referrers by click volume.
type ReferrerCount struct {
	Referrer string `json:"referrer" ch:"referrer"`
	Clicks   uint64 `json:"clicks"   ch:"clicks"`
}
