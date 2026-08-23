package data

import (
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	RedisDataTTL = 24 * time.Hour  // redis data lifetime
	PgCtxTimeout = 3 * time.Second // pgx query request context lifetime
	ChCtxTimeout = 3 * time.Second // [23-08-2026] FIXME:
)

type Models struct {
	Cache        *RedisStore
	PostgresDB   *PostgresStore
	ClickhouseDB *ClickHouseStore
}

func NewModels(
	pgDB *pgxpool.Pool,
	chDB driver.Conn,
	rdb *redis.Client,
	logger *slog.Logger,
) Models {
	return Models{
		Cache:        NewRedisStore(rdb, RedisDataTTL, logger),
		PostgresDB:   NewPostgresStore(pgDB, PgCtxTimeout, logger),
		ClickhouseDB: NewClickHouseStore(chDB, ChCtxTimeout, logger),
	}
}

// [21-07-2026] DOUBT: How can we refactor the Models struct, accompanying
// both the Stores
