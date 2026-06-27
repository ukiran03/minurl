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
}

func Load() (*Config, error) {
	var cfg Config

	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(&cfg.Env, "env", "development",
		"Environment (development|staging|production)")
	flag.StringVar(
		&cfg.DB.DSN, "db-dsn",
		os.Getenv("MINURL_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db-max-open-conns",
		25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.DB.MaxIdleConns, "db-max-idle-conns",
		25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.DB.MaxIdleTime, "db-max-idle-time",
		2*time.Minute, "PostgreSQL max connection idle time")

	sfNodeStr := os.Getenv("SNOWFLAKE_NODE_ID")
	if sfNodeStr == "" {
		return nil, fmt.Errorf("SNOWFLAKE_NODE_ID environment variable is empty")
	}
	var err error
	cfg.SFNode, err = strconv.Atoi(sfNodeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SFNID (Snowflake Node ID) format: %v", err)
	}

	flag.Parse()

	return &cfg, nil
}
