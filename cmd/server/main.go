package main

import (
	"log"
	"net/http"

	"github.com/ayanosky/url-shortener2.0/internal/handler"
	"github.com/ayanosky/url-shortener2.0/internal/service"
	"github.com/ayanosky/url-shortener2.0/internal/storage"
)

func main() {
	store := storage.NewMemory()
	svc := service.New(store)
	h := handler.New(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /shorten", h.Shorten)
	mux.HandleFunc("GET /{code}", h.Redirect)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
