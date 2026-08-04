package models

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	DB      *pgxpool.Pool
	Timeout time.Duration // operational timeout
	Logger  *slog.Logger
}

func NewPostgresStore(
	db *pgxpool.Pool, timeout time.Duration, logger *slog.Logger,
) *PostgresStore {
	return &PostgresStore{
		DB:      db,
		Timeout: timeout,
		Logger:  logger, // just in case
	}
}

func (s *PostgresStore) Put(ctx context.Context, minurl *MinUrl) error {
	query := `INSERT INTO minurls (slug, url, created_at, expires_at)
	          VALUES ($1, $2, $3, $4)`
	params := []any{
		minurl.Flake, minurl.URL,
		minurl.Life.Created, minurl.Life.Expiry,
	}

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	_, err := s.DB.Exec(ctx, query, params...)
	// Assuming on Duplicate Record errors, because of our stateless
	// snowflake ID logic
	return err
}

func (s *PostgresStore) Get(
	ctx context.Context, minurl *MinUrl,
) (string, error) {
	query := `SELECT url FROM minurls WHERE slug = $1`

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	var longURL string
	err := s.DB.QueryRow(ctx, query, minurl.Flake).Scan(&longURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRecordNotFound
		}
		return "", err
	}
	return longURL, nil
}

func (s *PostgresStore) Delete(ctx context.Context, minurl *MinUrl) error {
	query := `DELETE FROM minurls WHERE slug = $1`

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	result, err := s.DB.Exec(ctx, query, minurl.Flake)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (s *PostgresStore) Copy(ctx context.Context, minurls []MinUrl) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	table := pgx.Identifier{"minurls"}
	columns := []string{"slug", "url", "created_at", "expires_at"}

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	_, err = tx.CopyFrom(ctx,
		table,
		columns,
		pgx.CopyFromSlice(len(minurls), func(i int) ([]any, error) {
			m := minurls[i]
			return []any{m.Flake, m.URL, m.Life.Created, m.Life.Expiry}, nil
		}),
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
