package data

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
	query := `INSERT INTO minurls (slug, url,url_hash, created_at, expires_at)
	          VALUES ($1, $2, $3, $4, $5)`
	params := []any{
		minurl.Flake, minurl.URL, minurl.URLHash,
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
	query := `
		SELECT url
		FROM minurls
		WHERE slug = $1
		  AND (expires_at IS NULL OR expires_at > NOW())
	`

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

// GetByHashWithAll fetches the original slug, long URL, and expiry (almost all
// fields) using the MD5 hash.  It automatically ignores expired records.
func (s *PostgresStore) GetByHashWithAll(
	ctx context.Context,
	urlHash string,
) (int64, string, *time.Time, error) {
	query := `
		SELECT slug, url, expires_at
		FROM minurls
		WHERE url_hash = $1
		  AND (expires_at IS NULL OR expires_at > NOW())
	`
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	var (
		slugInt int64
		longURL string
		expiry  *time.Time
	)

	err := s.DB.QueryRow(ctx, query, urlHash).Scan(&slugInt, &longURL, &expiry)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", nil, ErrRecordNotFound
		}
		return 0, "", nil, err
	}
	return slugInt, longURL, expiry, nil
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
	if len(minurls) == 0 {
		return nil
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	// Create a unconstrained temporary staging table for the transaction
	_, err = tx.Exec(ctx, `
		CREATE TEMP TABLE minurls_staging (
			LIKE minurls INCLUDING DEFAULTS
		) ON COMMIT DROP;
	`)
	if err != nil {
		return err
	}

	// Stream batch into the staging table (now will never fail on unique constraints)
	table := pgx.Identifier{"minurls_staging"}
	columns := []string{"slug", "url", "url_hash", "created_at", "expires_at"}

	_, err = tx.CopyFrom(ctx,
		table,
		columns,
		pgx.CopyFromSlice(len(minurls), func(i int) ([]any, error) {
			m := minurls[i]
			return []any{
				m.Flake,
				m.URL,
				m.URLHash,
				m.Life.Created,
				m.Life.Expiry,
			}, nil
		}),
	)
	if err != nil {
		return err
	}

	// Upsert from minurls_staging into the real table, ignoring duplicate
	// hashes/slugs safely
	_, err = tx.Exec(ctx, `
		INSERT INTO minurls (slug, url, url_hash, created_at, expires_at)
		SELECT slug, url, url_hash, created_at, expires_at
		FROM minurls_staging
		ON CONFLICT (url_hash) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)

	// [24-08-2026] NOTE: Cannot use COPY for upserts because PostgreSQL's
	// native COPY FROM command does not support ON CONFLICT clauses.
}

// [25-08-2026] TODO: setup `pg_cron` background job for BATCH deleting expired
// records, in the manifest files, to deploy on K3d (later GKE)
