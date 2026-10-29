package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/ayanosky/url-shortener2.0/internal/handler"
	"github.com/ayanosky/url-shortener2.0/internal/service"
	"github.com/ayanosky/url-shortener2.0/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	store := storage.NewPostgres(pool)
	svc := service.New(store)
	h := handler.New(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /shorten", h.Shorten)
	mux.HandleFunc("GET /{code}", h.Redirect)
	mux.HandleFunc("GET /stats/{code}", h.Stats)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
