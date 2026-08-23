package data

import (
	"net"
	"net/http"
	"strings"
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

type ClickEvent struct {
	Slug       string    `json:"slug"`
	Timestamp  time.Time `json:"timestamp"`
	RemoteAddr string    `json:"remote_addr"`
	UserAgent  string    `json:"user_agent"`
	Referrer   string    `json:"referrer"`
}

func NewClickEvent(slug string, r *http.Request) *ClickEvent {
	ip := extractIP(r)

	return &ClickEvent{
		Slug:       slug,
		Timestamp:  time.Now().UTC(),
		RemoteAddr: ip,
		UserAgent:  r.UserAgent(),
		Referrer:   r.Referer(),
	}
}

// Helper to reliably extract the client IP
func extractIP(r *http.Request) string {
	// check for standard proxy headers first
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For can contain a comma-separated list of IPs (client,
		// proxy1, proxy2)
		parts := strings.Split(fwd, ",")
		if clientIP := strings.TrimSpace(parts[0]); clientIP != "" {
			return clientIP
		}
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// fall back to RemoteAddr, making sure to strip the port
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If RemoteAddr didn't have a port for some reason, return it as-is
		return r.RemoteAddr
	}

	return ip
}
