package models

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MinUrl struct {
	Slug      string             `db:"slug"`
	Name      string             `db:"name"`
	URL       string             `db:"url"`
	OwnerID   pgtype.UUID        `db:"owner_id"`
	CreatedAt time.Time          `db:"created_at"`
	ExpiresAt pgtype.Timestamptz `db:"expires_at"`
	IsCustom  bool               `db:"is_custom"`
}

type MinUrlModel struct {
	Pool *pgxpool.Pool
}

func (m *MinUrlModel) Latest(ctx context.Context) ([]MinUrl, error) {
	stmt := `SELECT slug, COALESCE(name, slug) AS name, url,
             owner_id, created_at, expires_at, is_custom
             FROM minurls ORDER BY created_at DESC
             LIMIT 10`

	rows, err := m.Pool.Query(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("querying latest urls: %w", err)
	}
	murls, err := pgx.CollectRows(rows, pgx.RowToStructByName[MinUrl])
	if err != nil {
		return nil, fmt.Errorf("collecting rows: %w", err)
	}
	return murls, nil
}

func (mu MinUrl) String() string {
	return fmt.Sprintf(
		"%s\n%s\n%s\n%s\n\n",
		mu.Slug, mu.URL, mu.OwnerID, mu.CreatedAt,
	)
}
