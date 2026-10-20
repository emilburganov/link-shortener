package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emilburganov/link-shortener/internal/config"
	"github.com/emilburganov/link-shortener/internal/handler"
	"github.com/emilburganov/link-shortener/internal/link"
	"github.com/emilburganov/link-shortener/internal/repository/postgres"
)

const (
	dbConnectTimeout = 5 * time.Second
	shutdownTimeout  = 8 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()

	if err != nil {
		return err
	}

	gin.SetMode(cfg.GinMode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := connectDB(ctx, cfg.DatabaseURL)

	if err != nil {
		return err
	}

	defer pool.Close()

	slog.Info("connected to database")

	links := link.NewService(postgres.NewRepository(pool))
	router := handler.New(links, cfg.BaseURL, cfg.FrontendURL).Routes()

	return serve(ctx, newServer(cfg.Port, router))
}

func connectDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)

	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, dbConnectTimeout)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}

func newServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func serve(ctx context.Context, srv *http.Server) error {
	slog.Info("server starting", "addr", srv.Addr)

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
