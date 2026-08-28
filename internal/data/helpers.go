package data

import (
	"net"
	"net/http"
	"strings"
	"unicode"
)

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

func sanitizeReferrer(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "Direct"
	}

	// Check for any control characters
	for _, r := range ref {
		if unicode.IsControl(r) {
			return "Direct"
		}
	}

	return ref
}
