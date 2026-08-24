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
}

func Load() (*Config, error) {
	var cfg Config

	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(&cfg.Env, "env", "development",
		"Environment (development|staging|production)")

	// -- Postgres
	flag.StringVar(
		&cfg.DB.DSN, "db-dsn",
		os.Getenv("POSTGRES_DSN"), "PostgreSQL DSN",
	)
	flag.IntVar(
		&cfg.DB.MaxOpenConns, "db-max-open-conns",
		25, "PostgreSQL max open connections",
	)
	flag.IntVar(
		&cfg.DB.MaxIdleConns, "db-max-idle-conns",
		25, "PostgreSQL max idle connections",
	)
	flag.DurationVar(
		&cfg.DB.MaxIdleTime, "db-max-idle-time",
		2*time.Minute, "PostgreSQL max connection idle time",
	)

	// -- ClickHouse
	flag.StringVar(
		&cfg.CHDB.Host, "chdb-host",
		os.Getenv("CLICKHOUSE_HOST"), "ClickHouse Host name",
	)
	flag.IntVar(&cfg.CHDB.Port, "chdb-port",
		9000, "ClickHouse DB port")
	flag.StringVar(
		&cfg.CHDB.DbName, "chdb-db",
		os.Getenv("CLICKHOUSE_DB"), "ClickHouse DB name",
	)
	flag.StringVar(
		&cfg.CHDB.Username, "chdb-user",
		os.Getenv("CLICKHOUSE_USER"), "ClickHouse Username",
	)
	flag.StringVar(
		&cfg.CHDB.Passwd, "chdb-passwd", // Fixed from DbName to Password
		os.Getenv("CLICKHOUSE_PASSWD"), "ClickHouse DB Password",
	)

	// -- Redis
	flag.StringVar(
		&cfg.RDB.Addr, "rdb-addr", os.Getenv("REDIS_ADDR"), "Redis Address",
	)
	flag.StringVar(
		&cfg.RDB.Passwd,
		"rdb-passwd", os.Getenv("REDIS_PASSWD"), "Redis Password",
	)
	// -- Nats.io
	flag.StringVar(
		&cfg.NATS.URL, "nats-url", os.Getenv("NATS_URL"), "NATS URL",
	)

	// -- Snowflake
	sfNodeStr := os.Getenv("SNOWFLAKE_NODE_ID")
	if sfNodeStr == "" {
		return nil, fmt.Errorf(
			"SNOWFLAKE_NODE_ID environment variable is empty",
		)
	}

	var err error
	cfg.SFNode, err = strconv.Atoi(sfNodeStr)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid SFNID (Snowflake Node ID) format: %v",
			err,
		)
	}

	flag.Parse()

	return &cfg, nil
}
