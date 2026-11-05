package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ayanosky/url-shortener2.0/internal/service"
	"github.com/ayanosky/url-shortener2.0/internal/storage"
)

func newTestMux() *http.ServeMux {
	h := New(service.New(storage.NewMemory()))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /shorten", h.Shorten)
	mux.HandleFunc("GET /stats/{code}", h.Stats)
	mux.HandleFunc("GET /{code}", h.Redirect)
	return mux
}

func TestShorten_BadJSON(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader("abc"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if body := rec.Body.String(); !strings.Contains(body, "invalid json") {
		t.Errorf("body = %q, want it to contain %q", body, "invalid json")
	}
}

func TestShorten_InvalidURL(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"ftp://x.com"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if body := rec.Body.String(); !strings.Contains(body, "invalid url") {
		t.Errorf("body = %q, want it to contain %q", body, "invalid url")
	}
}

func TestRedirect_NotFound(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestStats_NotFound(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/stats/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHealth(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func do(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestFullFlow(t *testing.T) {
	mux := newTestMux()

	rec := do(t, mux, http.MethodPost, "/shorten", `{"url":"https://go.dev"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("shorten status = %d, want %d", rec.Code, http.StatusCreated)
	}
	var created shortenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal shorten response: %v", err)
	}

	rec = do(t, mux, http.MethodGet, "/"+created.Code, "")
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != "https://go.dev" {
		t.Errorf("Location = %q, want %q", got, "https://go.dev")
	}

	rec = do(t, mux, http.MethodGet, "/stats/"+created.Code, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want %d", rec.Code, http.StatusOK)
	}
	var stats statsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("unmarshal stats response: %v", err)
	}
	if stats.Clicks != 1 {
		t.Errorf("clicks = %d, want 1", stats.Clicks)
	}
}

func TestShorten_MethodNotAllowed(t *testing.T) {
	mux := newTestMux()

	rec := do(t, mux, http.MethodDelete, "/shorten", "")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "POST") {
		t.Errorf("Allow = %q, want it to contain POST", allow)
	}
}
