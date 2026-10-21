package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code string `json:"code"`
}

// Хэндлеры вызываются из разных горутин, поэтому map надо защищать мьютексом
var (
	mu    sync.RWMutex
	links = map[string]string{}
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /shorten", shortenHandler)
	mux.HandleFunc("GET /{code}", redirectHandler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// TODO: проверить, что URL не пустой и валидный (net/url)
	// TODO: сгенерировать случайный код вместо заглушки
	code := "abc123"

	mu.Lock()
	links[code] = req.URL
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shortenResponse{Code: code})
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	mu.Lock()
	url, ok := links[code]
	mu.Unlock()

	if !ok {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}
