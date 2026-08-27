package data

import (
	"context"
	"errors"
	"fmt"
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

	// [27-08-2026] NOTE: here the `arrayMap` is similar to
	// Lisp's map:  `(map proc lst ...+) → list?`
	query := fmt.Sprintf(`
    SELECT
        COUNT() AS total_clicks,
        arrayMap(x -> (x.1, x.2), topK(%d)(referrer)) AS top_referrers
    FROM url_clicks
    WHERE slug = ? AND TIMESTAMP >= ? AND TIMESTAMP < ?
`, limit)

	stats := &ClickStats{
		Slug:         slug,
		From:         from,
		To:           to,
		TopReferrers: make([]ReferrerCount, 0, limit),
	}

	row := s.DB.QueryRow(ctx, query, slug, from, to)

	// ClickHouse driver maps topK array of tuples to custom Go slices/types
	var topRefTuples []struct {
		Referrer string `ch:"1"`
		Clicks   uint64 `ch:"2"`
	}

	if err := row.Scan(&stats.TotalClicks, &topRefTuples); err != nil {
		return nil, err
	}

	for _, t := range topRefTuples {
		stats.TopReferrers = append(stats.TopReferrers, ReferrerCount{
			Referrer: t.Referrer,
			Clicks:   int64(t.Clicks),
		})
	}

	return stats, nil
}
