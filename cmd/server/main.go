package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
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

var ErrInvalidURL = errors.New("invalid url")

func ValidateURL(rawURL string) error {
	// ParseRequestURI требует, чтобы строка была абсолютным URI,
	// то есть обязательно начиналась со схемы (например, http://)
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("%w: malformed url", ErrInvalidURL)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https", ErrInvalidURL)
	}

	if u.Host == "" {
		return fmt.Errorf("%w: host is empty", ErrInvalidURL)
	}

	return nil
}

func generateCodeFast(n int) string {
	b := make([]byte, n)
	const (
		alphabet     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		letterIdBits = 6                   // 6 бит на символ (2^6 = 64)
		letterIdMask = 1<<letterIdBits - 1 // Маска 00111111 (63)
		letterIdMax  = 63 / letterIdBits   // Сколько индексов умещается в одном uint64 (10)
	)

	for i, cache, remain := n-1, rand.Uint64(), letterIdMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Uint64(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(alphabet) {
			b[i] = alphabet[idx]
			i--
		}
		cache >>= letterIdBits
		remain--
	}
	return string(b)
}

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

	if err := ValidateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code := generateCodeFast(6)

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
