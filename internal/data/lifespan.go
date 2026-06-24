package data

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"
)

var ErrInvalidExpiryFormat = errors.New(
	"invalid expires_at value: must be 1d, 1w, 1m, 1y, or empty",
)

type Lifespan struct {
	Created time.Time `json:"created_at"`
	Expiry  time.Time `json:"expires_at"`
}

func (l *Lifespan) UnmarshalJSON(data []byte) error {
	// define an alias type to prevent infinite recursion duing Unmarshal
	type Alias Lifespan
	aux := &struct {
		// captures user string "1d", "1w", etc. using pointer string to
		// reliably detect missing vs empty keys
		Expiry *string `json:"expires_at"`
		*Alias
	}{
		Alias: (*Alias)(l),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	now := time.Now()
	l.Created = now

	// Fallback default if "expires_at" is missing entirely from the JSON
	if aux.Expiry == nil {
		l.Expiry = now.AddDate(0, 0, 7) // Default 1w
		return nil
	}

	var expiryTime time.Time
	durationInput := strings.ToLower(strings.TrimSpace(*aux.Expiry))

	switch durationInput {
	case "1d":
		expiryTime = now.AddDate(0, 0, 1)
	case "1m":
		expiryTime = now.AddDate(0, 1, 0)
	case "1y":
		expiryTime = now.AddDate(1, 0, 0)
	case "1w", "":
		// Handles explicit "1w" or explicit JSON empty string `""`
		expiryTime = now.AddDate(0, 0, 7)
	default:
		return ErrInvalidExpiryFormat
	}

	// Add high-throughput jitter (1-5 minutes) to smooth out database/cache
	// cleanup waves
	jitter := time.Duration(rand.Intn(240)+60) * time.Second
	l.Expiry = expiryTime.Add(jitter)

	return nil
}
