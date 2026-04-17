package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ukiran.com/minurl/internal/models"
)

type application struct {
	logger   *slog.Logger
	murls    *models.MinUrlModel
	users    *models.UserModel
	clickEvs *models.ClickEvModel
}

func main() {
	addr := flag.String("addr", ":3090", "HTTP network address")
	dsn := flag.String(
		"dsn",
		"postgres://ukiran:ukiran@localhost:5432/minurldb?sslmode=disable",
		"PostgreSQL data source name",
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx := context.Background()
	pool, err := openDB(ctx, *dsn)
	if err != nil {
		logger.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	app := application{
		logger:   logger,
		murls:    &models.MinUrlModel{Pool: pool, Ctx: ctx},
		users:    &models.UserModel{Pool: pool, Ctx: ctx},
		clickEvs: &models.ClickEvModel{Pool: pool, Ctx: ctx},
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}
	srv := &http.Server{
		Addr:         *addr,
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	logger.Info("starting server", "addr", *addr)

	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	// Verify the connection is actually alive
	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
