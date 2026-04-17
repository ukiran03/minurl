package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// represents a Click Event
type ClickEv struct {
	ID          int64     `db:"id"`
	Slug        string    `db:"slug"`
	ClickedAt   time.Time `db:"clicked_at"`
	IpAddress   string    `db:"ip_address"`
	UserAgent   string    `db:"user_agent"`
	Referrer    string    `db:"referrer"`
	CountryCode string    `db:"country_code"`
	DeviceType  string    `db:"device_type"`
}

type ClickEvModel struct {
	Pool *pgxpool.Pool
	Ctx  context.Context
}
