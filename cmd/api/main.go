package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"ukiran.com/minurl/internal/config"
	"ukiran.com/minurl/internal/data"
	"ukiran.com/minurl/internal/logger"
)

const version = "1.0.0"

type application struct {
	config *config.Config
	logger *slog.Logger
	models data.Models
}

func main() {
	logger := logger.NewLogger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Initialization error: %v", err)
	}

	db, err := openDB(cfg)
	if err != nil {
		logger.Error("unable to connect to database", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(
			db, cfg.SFNode,
		),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("starting server", "addr", srv.Addr, "env", cfg.Env)
	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
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
