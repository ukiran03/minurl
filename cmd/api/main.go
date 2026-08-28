package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
	"ukiran.com/minurl/internal/config"
	"ukiran.com/minurl/internal/data"
	"ukiran.com/minurl/internal/logger"
	"ukiran.com/minurl/internal/stream"
)

const version = "1.0.0"

type application struct {
	config           *config.Config
	logger           *slog.Logger
	models           data.Models
	postgresStream   stream.Streamer
	clickhouseStream stream.Streamer
	bloom            *data.BloomFilter
	sfid             int
	wg               sync.WaitGroup
}

func main() {
	if err := run(); err != nil {
		slog.Error("fatal application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Root context used to signal cancellation across the entire application
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	logger := logger.NewLogger()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	pgDB, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("unable to connect to postgres database: %w", err)
	}
	defer pgDB.Close()
	logger.Info("postgres database connection pool established")

	chDB, err := connectClickHouse(cfg) // [23-08-2026] TODO: Use config.Config
	if err != nil {
		return fmt.Errorf("unable to connect to clickhouse database: %w", err)
	}
	defer chDB.Close()
	logger.Info("clickhouse database connection pool established")

	rdb, err := connectRedis(cfg)
	if err != nil {
		return fmt.Errorf("unable to connect to redis: %w", err)
	}
	defer rdb.Close()
	logger.Info("redis cache connection pool established")

	bloom := data.NewBloomFilter(rdb)
	if err := bloom.InitFilter(appCtx); err != nil {
		return fmt.Errorf("unable to initialize bloom filter: %w", err)
	}

	nc, err := connectNats(cfg)
	if err != nil {
		return fmt.Errorf("nats connection error: %w", err)
	}
	// Close NATS connection when run() returns, AFTER wg.Wait() completes
	defer nc.Close()

	jets, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("jetStream error: %w", err)
	}

	models := data.NewModels(pgDB, chDB, rdb, logger)

	pgStream, err := stream.NewPostgresStream(
		appCtx,
		jets,
		models.PostgresDB,
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create postgres stream: %w", err)
	}

	chStream, err := stream.NewClickHouseStream(
		appCtx,
		jets,
		models.ClickhouseDB,
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create clickhouse stream: %w", err)
	}

	app := &application{
		config:           cfg,
		logger:           logger,
		models:           models,
		postgresStream:   pgStream,
		clickhouseStream: chStream,
		bloom:            bloom,
		sfid:             cfg.SFNode,
	}

	app.wg.Go(func() {
		if err := pgStream.Start(appCtx); err != nil &&
			!errors.Is(err, context.Canceled) {
			logger.Error(
				"postgers stream processor exited with error", "error", err,
			)
		}
	})

	app.wg.Go(func() {
		if err := chStream.Start(appCtx); err != nil &&
			!errors.Is(err, context.Canceled) {
			logger.Error(
				"clickhouse stream processor exited with error", "error", err,
			)
		}
	})

	// serve blocks until an interrupt signal is caught
	if err := app.serve(appCancel); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	// Wait for background workers (like pgStream) to finish before defers execute
	app.wg.Wait()
	logger.Info("application shutdown complete")
	return nil
}

func openDB(cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DB.DSN)
	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = int32(cfg.DB.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.DB.MaxIdleConns)
	poolCfg.MaxConnIdleTime = cfg.DB.MaxIdleTime
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, err
}

func connectClickHouse(cfg *config.Config) (driver.Conn, error) {
	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{
			fmt.Sprintf("%s:%d", cfg.CHDB.Host, cfg.CHDB.Port),
		},
		Auth: clickhouse.Auth{
			Database: cfg.CHDB.DbName,
			Username: cfg.CHDB.Username,
			Password: "",
		},
		DialTimeout: 5 * time.Second,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	})
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			return nil, fmt.Errorf(
				"clickhouse exception [%d]: %s",
				exception.Code,
				exception.Message,
			)
		}
		return nil, fmt.Errorf("failed to connect/ping clickhouse: %w", err)
	}

	return conn, nil
}

func connectRedis(cfg *config.Config) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:         cfg.RDB.Addr,
		Password:     cfg.RDB.Passwd,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	fmt.Printf(
		"Connecting to Redis at: [%s] with password length: %d\n",
		cfg.RDB.Addr,
		len(cfg.RDB.Passwd),
	)
	rdb := redis.NewClient(opts)

	var err error
	maxRetries := 15
	backoff := 1 * time.Second

	for i := range maxRetries {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = rdb.Ping(ctx).Err()
		cancel()

		if err == nil {
			log.Println("Successfully connected to Redis Stack!")
			return rdb, nil
		}

		log.Printf(
			"redis not ready yet (attempt %d/%d): %v. retrying in %v...",
			i+1,
			maxRetries,
			err,
			backoff,
		)

		time.Sleep(backoff)

		// Exponential backoff capped at 5 seconds
		backoff *= 2
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
	}

	_ = rdb.Close()
	return nil, fmt.Errorf("redis connection failed after retries: %w", err)
}

func connectNats(cfg *config.Config) (*nats.Conn, error) {
	natsURL := cfg.NATS.URL
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	return nc, err
}
