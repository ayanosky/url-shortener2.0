package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ayanosky/url-shortener2.0/internal/config"
	"github.com/ayanosky/url-shortener2.0/internal/handler"
	"github.com/ayanosky/url-shortener2.0/internal/service"
	"github.com/ayanosky/url-shortener2.0/internal/storage"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.SetDefault(newLogger(cfg.LogLevel))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	store := storage.NewPostgres(pool)
	svc := service.New(store)
	h := handler.New(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /shorten", h.Shorten)
	mux.HandleFunc("GET /stats/{code}", h.Stats)
	mux.HandleFunc("GET /{code}", h.Redirect)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		// сервер упал сам (например, порт занят)
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.Info("server stopped")
	return nil
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	// при неверном значении ошибка игнорируется, остаётся уровень по умолчанию (info)
	_ = lvl.UnmarshalText([]byte(level))

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
