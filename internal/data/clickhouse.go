package data

import (
	"context"
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
