package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port   int
	Env    string
	SFNode int
	DB     struct {
		DSN          string
		MaxOpenConns int
		MaxIdleConns int
		MaxIdleTime  time.Duration
	}
	CHDB struct {
		Host     string
		Port     int
		DbName   string
		Username string
		Passwd   string
	}
	RDB struct {
		Addr   string
		Passwd string
	}
	NATS struct {
		URL string
	}
	BaseURL string
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func Load() (*Config, error) {
	var cfg Config

	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(
		&cfg.Env,
		"env",
		"development",
		"Environment (development|staging|production)",
	)

	// -- Postgres
	flag.StringVar(
		&cfg.DB.DSN,
		"db-dsn",
		getEnv("POSTGRES_DSN", ""),
		"PostgreSQL DSN",
	)
	flag.IntVar(
		&cfg.DB.MaxOpenConns,
		"db-max-open-conns",
		25,
		"PostgreSQL max open connections",
	)
	flag.IntVar(
		&cfg.DB.MaxIdleConns,
		"db-max-idle-conns",
		25,
		"PostgreSQL max idle connections",
	)
	flag.DurationVar(
		&cfg.DB.MaxIdleTime,
		"db-max-idle-time",
		2*time.Minute,
		"PostgreSQL max connection idle time",
	)

	// -- ClickHouse
	flag.StringVar(
		&cfg.CHDB.Host,
		"chdb-host",
		getEnv("CLICKHOUSE_HOST", ""),
		"ClickHouse Host name",
	)
	flag.IntVar(&cfg.CHDB.Port, "chdb-port", 9000, "ClickHouse DB port")
	flag.StringVar(
		&cfg.CHDB.DbName,
		"chdb-chdb",
		getEnv("CLICKHOUSE_DB", ""),
		"ClickHouse DB name",
	)
	flag.StringVar(
		&cfg.CHDB.Username,
		"chdb-user",
		getEnv("CLICKHOUSE_USER", ""),
		"ClickHouse Username",
	)
	flag.StringVar(
		&cfg.CHDB.Passwd,
		"chdb-passwd",
		getEnv("CLICKHOUSE_PASSWD", ""),
		"ClickHouse DB Password",
	)

	// -- Redis
	flag.StringVar(
		&cfg.RDB.Addr,
		"rdb-addr",
		getEnv("REDIS_ADDR", ""),
		"Redis Address",
	)
	flag.StringVar(
		&cfg.RDB.Passwd,
		"rdb-passwd",
		getEnv("REDIS_PASSWD", ""),
		"Redis Password",
	)

	// -- Nats.io
	flag.StringVar(
		&cfg.NATS.URL,
		"nats-url",
		getEnv("NATS_URL", ""),
		"NATS URL",
	)

	// -- Base URL (handled dynamically after parsing)
	defaultBaseURL := getEnv("BASE_URL", "")
	flag.StringVar(
		&cfg.BaseURL,
		"base-url",
		defaultBaseURL,
		"Base URL for short links",
	)

	flag.Parse()

	if cfg.BaseURL == "" {
		cfg.BaseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
	}

	// -- Snowflake Validation
	sfNodeStr := getEnv("SNOWFLAKE_NODE_ID", "")
	if sfNodeStr == "" {
		return nil, fmt.Errorf(
			"SNOWFLAKE_NODE_ID environment variable is empty",
		)
	}

	snowId, err := strconv.Atoi(sfNodeStr)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid SFNID (Snowflake Node ID) format: %w",
			err,
		)
	}
	if snowId < 0 || snowId > 1023 {
		return nil, fmt.Errorf("invalid Snowflake node IDs (must be 0–1023)")
	}
	cfg.SFNode = snowId

	return &cfg, nil
}
