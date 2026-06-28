package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicateSlug = errors.New("duplicate slug")

type MinUrl struct {
	Slug   string   `json:"slug"` // Holds "9Gf3xZ" (Base62) OR "my-custom-slug"
	URL    string   `json:"url"`
	Title  *string  `json:"title,omitzero"`
	UserID *int64   `json:"user_id,omitzero"`
	Life   Lifespan `json:"lifespan"`
}

type MinUrlModel struct {
	SFNID int64 // Snowflake Node ID
	DB    *pgxpool.Pool
	TTL   time.Duration
}

func (m MinUrlModel) Insert(ctx context.Context, minurl *MinUrl) error {
	query := `INSERT INTO minurls (slug, url, title, user_id, created_at, expires_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	snowflake := NewFlake(m.SFNID)
	if err := m.executeInsert(ctx, query, snowflake, minurl); err != nil {
		return err
	}
	// Mutate the struct Convert that int64 to Base62 string
	minurl.Slug = snowflake.Base62()
	return nil
}

func (m MinUrlModel) InsertCustom(ctx context.Context, minurl *MinUrl) error {
	if minurl.UserID == nil {
		return fmt.Errorf("user_id is required for custom URLs")
	}

	query := `INSERT INTO custom_minurls
              (slug, url, title, user_id, created_at, expires_at)
              VALUES ($1, $2, $3, $4, $5, $6)`
	return m.executeInsert(ctx, query, minurl.Slug, minurl)
}

func (m MinUrlModel) executeInsert(
	ctx context.Context, query string,
	slug any, minurl *MinUrl,
) error {
	params := []any{
		slug,
		minurl.URL, minurl.Title, minurl.UserID,
		minurl.Life.Created, minurl.Life.Expiry,
	}
	ctx, cancel := context.WithTimeout(ctx, m.TTL)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, params...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrDuplicateSlug
		}
		return err
	}
	return nil
}

// GetMinUrl will just give the URL to redirect
func (m MinUrlModel) GetMinUrl(ctx context.Context, slug string) (string, error) {
	snowflake, err := ParseBase62(slug)
	if err != nil {
		return "", err
	}
	query := `SELECT url FROM minurls WHERE slug = $1`
	return m.executeGetMinUrl(ctx, query, snowflake)
}

func (m MinUrlModel) GetMinUrlCustom(ctx context.Context, slug string) (string, error) {
	query := `SELECT url FROM custom_minurls WHERE slug = $1`
	return m.executeGetMinUrl(ctx, query, slug)
}

func (m MinUrlModel) executeGetMinUrl(
	ctx context.Context, query string, slug any,
) (string, error) {
	params := []any{slug}

	ctx, cancel := context.WithTimeout(ctx, m.TTL)
	defer cancel()

	var url string
	err := m.DB.QueryRow(ctx, query, params...).Scan(&url)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRecordNotFound
		}
		return "", err
	}
	return url, nil
}

func (m MinUrlModel) Get(slug string) error {
	return nil
}

func (m MinUrlModel) Delete(slug string) error {
	return nil
}
