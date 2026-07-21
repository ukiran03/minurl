package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

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
	config *config.Config
	logger *slog.Logger
	models data.Models
	stream stream.Streamer
	bloom  *data.BloomFilter
	sfid   int
	wg     sync.WaitGroup
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

	db, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	rdb, err := connectRedis("localhost:6379", "") // TODO: get via config
	if err != nil {
		return fmt.Errorf("unable to connect to redis: %w", err)
	}
	defer rdb.Close()
	logger.Info("redis cache connection pool established")

	bloom := data.NewBloomFilter(rdb)
	if err := bloom.InitFilter(appCtx); err != nil {
		return fmt.Errorf("unable to initialize bloom filter: %w", err)
	}

	jets, err := connectNats(nats.DefaultURL)
	if err != nil {
		return fmt.Errorf("jetStream error: %w", err)
	}

	models := data.NewModels(db, rdb, logger)

	pgStream, err := stream.NewPostgresStream(appCtx, jets, models.DB)
	if err != nil {
		return fmt.Errorf("failed to create postgres stream: %w", err)
	}

	app := &application{
		config: cfg,
		logger: logger,
		models: models,
		stream: pgStream,
		bloom:  bloom,
		sfid:   cfg.SFNode,
	}

	app.wg.Go(func() {
		if err := pgStream.Start(appCtx); err != nil &&
			!errors.Is(err, context.Canceled) {
			logger.Error("stream processor exited with error", "error", err)
		}
	})

	// serve blocks until an interrupt signal is caught
	if err := app.serve(appCancel); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	// Wait for background workers (like pgStream) to finish before defers
	// close DB/Redis
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

func connectRedis(addr, passwd string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     passwd,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// shorter timeout for the initial Ping
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		// Clean up the client if the connection is dead
		_ = rdb.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return rdb, nil
}

func connectNats(natsURL string) (jetstream.JetStream, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	defer nc.Close()

	return jetstream.New(nc)
}
