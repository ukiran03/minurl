package data

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouseStore struct {
	DB      driver.Conn
	Timeout time.Duration // operational timeout
	Logger  *slog.Logger
}

func NewClickHouseStore(
	db driver.Conn,
	timeout time.Duration,
	logger *slog.Logger,
) *ClickHouseStore {
	return &ClickHouseStore{
		DB:      db,
		Timeout: timeout,
		Logger:  logger,
	}
}

func (s *ClickHouseStore) Write(
	ctx context.Context,
	clickEvs []ClickEvent,
) error {
	if len(clickEvs) == 0 {
		return nil
	}

	batch, err := s.DB.PrepareBatch(
		ctx,
		"INSERT INTO url_clicks (slug, timestamp, remote_addr, user_agent, referrer)",
	)
	if err != nil {
		return err
	}

	// Ensure the batch is cleaned, if an error occurs before Send() succeeds.
	// A successful Send() automatically closes the batch, so we use the 'sent'
	// flag to prevent redundant cleanup in the defer block.
	// Prefer close-on-error-only.
	var sent bool
	defer func() {
		if !sent {
			if err := batch.Close(); err != nil {
				s.Logger.Error("failed to close batch", "error", err)
			}
		}
	}()

	for _, event := range clickEvs {
		if err := batch.AppendStruct(&event); err != nil {
			return err
		}
	}

	if err := batch.Send(); err != nil {
		return err
	}

	// Mark as successfully sent to skip the cleanup logic in the defer func
	sent = true
	return nil
}

func (s *ClickHouseStore) GetClickStats(
	ctx context.Context,
	slug string,
	from, to time.Time,
	limit int,
) (*ClickStats, error) {
	const maxLimit = 65536
	if limit > maxLimit {
		return nil, errors.New("limit cannot exceed 65536")
	}

	stats := &ClickStats{
		Slug:         slug,
		From:         from,
		To:           to,
		TopReferrers: make([]ReferrerCount, 0, limit),
	}

	// Get total clicks for the slug in the time range
	totalQuery := `
		SELECT count()
		FROM url_clicks
		WHERE slug = ? AND timestamp >= ? AND timestamp < ?
	`
	if err := s.DB.QueryRow(ctx, totalQuery, slug, from, to).
		Scan(&stats.TotalClicks); err != nil {
		return nil, err
	}

	// If there are no clicks, we can skip the second query
	if stats.TotalClicks == 0 {
		return stats, nil
	}

	// Get top referrers
	topQuery := `
		SELECT
			referrer,
			COUNT() AS count
		FROM url_clicks
		WHERE slug = ? AND timestamp >= ? AND timestamp < ?
		GROUP BY referrer
		ORDER BY count DESC
		LIMIT ?
	`
	rows, err := s.DB.Query(ctx, topQuery, slug, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ref string
		var clicks uint64
		if err := rows.Scan(&ref, &clicks); err != nil {
			return nil, err
		}
		stats.TopReferrers = append(stats.TopReferrers, ReferrerCount{
			Referrer: ref,
			Clicks:   uint64(clicks),
		})
	}

	return stats, nil
}
