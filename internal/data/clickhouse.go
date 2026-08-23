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

	// Ensure the batch is aborted if we exit with an error before sending
	defer func() {
		if err := batch.Close(); err != nil {
			s.Logger.Error("failed to close batch", "error", err)
		}
	}()

	for _, event := range clickEvs {
		err := batch.Append(
			event.Slug,
			event.Timestamp,
			event.RemoteAddr,
			event.UserAgent,
			event.Referrer,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}
