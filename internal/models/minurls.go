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
	Name      string             `db:""`
	Slug      string             `db:"slug"`
	URL       string             `db:"url"`
	OwnerID   pgtype.UUID        `db:"owner_id"`
	CreatedAt time.Time          `db:"created_at"`
	ExpiresAt pgtype.Timestamptz `db:"expires_at"`
	IsCustom  bool               `db:"is_custom"`
}

type MinUrlModel struct {
	Pool *pgxpool.Pool
	Ctx  context.Context // TODO: ctx should not be in a struct, pass as a arg
}

func (m *MinUrlModel) Latest() ([]MinUrl, error) {
	stmt := `SELECT slug, url, owner_id, created_at, expires_at, is_custom
             FROM minurls ORDER BY created_at DESC
             LIMIT 10`

	rows, err := m.Pool.Query(m.Ctx, stmt)
	if err != nil {
		return nil, err
	}
	murls, err := pgx.CollectRows(rows, pgx.RowToStructByName[MinUrl])
	if err != nil {
		return nil, err
	}
	return murls, nil
}

func (mu MinUrl) String() string {
	return fmt.Sprintf(
		"%s\n%s\n%s\n%s\n\n",
		mu.Slug, mu.URL, mu.OwnerID, mu.CreatedAt,
	)
}
