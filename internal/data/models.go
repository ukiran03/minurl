package data

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	RedisDataTTL = 24 * time.Hour  // redis data lifetime
	PgCtxTimeout = 3 * time.Second // pgx query request context lifetime
)

type Models struct {
	Cache *RedisStore
	DB    *PostgresStore
}

func NewModels(
	db *pgxpool.Pool, rdb *redis.Client, logger *slog.Logger,
) Models {
	return Models{
		Cache: NewRedisStore(rdb, RedisDataTTL, logger),
		DB:    NewPostgresStore(db, PgCtxTimeout, logger),
	}
}

// [21-07-2026] DOUBT: How can we refactor the Models struct, accompanying
// both the Stores
