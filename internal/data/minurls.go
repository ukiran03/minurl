package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MinUrl struct {
	Slug   string   `json:"slug"` // Holds "9Gf3xZ" (Base58) OR "my-custom-slug"
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

	snowflakeID := NewSnowflakeID(m.SFNID)
	if err := m.execute(ctx, query, snowflakeID, minurl); err != nil {
		return err
	}
	// Mutate the struct Convert that int64 to Base58 string
	minurl.Slug = snowflakeID.Base58()
	return nil
}

func (m MinUrlModel) InsertCustom(ctx context.Context, minurl *MinUrl) error {
	if minurl.UserID == nil {
		return fmt.Errorf("user_id is required for custom URLs")
	}

	query := `INSERT INTO custom_minurls
              (slug, url, title, user_id, created_at, expires_at)
              VALUES ($1, $2, $3, $4, $5, $6)`

	return m.execute(ctx, query, minurl.Slug, minurl)
}

func (m MinUrlModel) execute(
	ctx context.Context, query string,
	slug interface{}, minurl *MinUrl,
) error {
	params := []any{
		slug,
		minurl.URL, minurl.Title, minurl.UserID,
		minurl.Life.Created, minurl.Life.Expiry,
	}
	ctx, cancel := context.WithTimeout(ctx, m.TTL)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, params...)
	return err
}

func (m MinUrlModel) Get(slug string) error {
	return nil
}

func (m MinUrlModel) Delete(slug string) error {
	return nil
}
