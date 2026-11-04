package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ayanosky/url-shortener2.0/internal/storage"
)

type collidingStore struct {
	storage.Storage     // встроенный интерфейс
	failFirst       int // сколько первых вызовов Save вернут коллизию
	calls           int // сколько раз вызвали Save
}

func (c *collidingStore) Save(ctx context.Context, code, url string) error {
	c.calls++
	if c.calls <= c.failFirst {
		return storage.ErrAlreadyExists
	}
	return c.Storage.Save(ctx, code, url)
}

func TestShorten_RetriesOnCollision(t *testing.T) {
	store := &collidingStore{Storage: storage.NewMemory(), failFirst: 2}
	svc := New(store)
	ctx := context.Background()

	code, err := svc.Shorten(ctx, "https://go.dev")
	if err != nil {
		t.Fatalf("Shorten: %v", err)
	}
	if store.calls != 3 {
		t.Errorf("Save calls = %d, want 3", store.calls)
	}

	// код действительно сохранился, а не потерялся
	if _, err := svc.Resolve(ctx, code); err != nil {
		t.Errorf("Resolve after retry: %v", err)
	}
}

func TestShorten_GivesUpAfterMaxRetries(t *testing.T) {
	store := &collidingStore{Storage: storage.NewMemory(), failFirst: 1000}
	svc := New(store)
	ctx := context.Background()

	_, err := svc.Shorten(ctx, "https://go.dev")
	if !errors.Is(err, ErrCodeGeneration) {
		t.Errorf("err = %v, want ErrCodeGeneration", err)
	}
	if store.calls != maxSaveRetries {
		t.Errorf("Save calls = %d, want %d", store.calls, maxSaveRetries)
	}
}

type dbDownStore struct {
	storage.Storage
	calls int
}

var errDBDown = errors.New("db down")

func (d *dbDownStore) Save(ctx context.Context, code, url string) error {
	d.calls++
	return errDBDown
}

func TestShorten_StopsOnStorageError(t *testing.T) {
	store := &dbDownStore{Storage: storage.NewMemory()}
	svc := New(store)
	ctx := context.Background()

	_, err := svc.Shorten(ctx, "https://go.dev")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, errDBDown) {
		t.Errorf("err = %v, want errDBDown", err)
	}
	if store.calls != 1 {
		t.Errorf("calls = %d, want 1", store.calls)
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://go.dev", false},
		{"valid http with path", "http://example.com/a?b=1", false},
		{"empty", "", true},
		{"no scheme", "go.dev", true},
		{"ftp scheme", "ftp://example.com", true},
		{"no host", "http://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func TestGenerateCode(t *testing.T) {
	for _, n := range []int{1, 6, 20} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			code := generateCode(n)
			if len(code) != n {
				t.Errorf("len = %d, want %d", len(code), n)
			}
			for _, c := range code {
				if !strings.ContainsRune(alphabet, c) {
					t.Errorf("unexpected char %q in %q", c, code)
				}
			}
		})
	}

	code1 := generateCode(6)
	code2 := generateCode(6)
	if code1 == code2 {
		t.Errorf("code1 equals code2")
	}
}

func TestGenerateCode_UsesWholeAlphabet(t *testing.T) {
	seen := make(map[rune]bool)
	for _, c := range generateCode(10000) {
		seen[c] = true
	}
	if len(seen) != len(alphabet) {
		t.Errorf("saw %d distinct chars, want %d", len(seen), len(alphabet))
	}
}

func TestShorten_InvalidURL(t *testing.T) {
	svc := New(storage.NewMemory())
	ctx := context.Background()

	_, err := svc.Shorten(ctx, "ftp://example.com")
	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("err = %v, want ErrInvalidURL", err)
	}
}

func TestShorten_ThenResolve(t *testing.T) {
	svc := New(storage.NewMemory())
	ctx := context.Background()

	code, err := svc.Shorten(ctx, "https://go.dev")
	if err != nil {
		t.Fatalf("Shorten: %v", err)
	}
	if len(code) != codeLength {
		t.Errorf("length = %v, want %v", len(code), codeLength)
	}

	rawURL, err := svc.Resolve(ctx, code)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if rawURL != "https://go.dev" {
		t.Errorf("url = %q, want 'https://go.dev'", rawURL)
	}

	linkStats, err := svc.Stats(ctx, code)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if linkStats.Clicks != 1 {
		t.Errorf("clicks = %v, want 1", linkStats.Clicks)
	}
}

func TestResolve_NotFound(t *testing.T) {
	svc := New(storage.NewMemory())
	ctx := context.Background()

	_, err := svc.Resolve(ctx, "abc123")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestStats_NotFound(t *testing.T) {
	svc := New(storage.NewMemory())
	ctx := context.Background()

	_, err := svc.Stats(ctx, "abc123")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
