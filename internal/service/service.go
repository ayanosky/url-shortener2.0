package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"

	"github.com/ayanosky/url-shortener2.0/internal/storage"
)

const (
	codeLength     = 6
	maxSaveRetries = 5
)

var (
	ErrInvalidURL = errors.New("invalid url")

	ErrCodeGeneration = errors.New("could not generate unique code")
)

func validateURL(rawURL string) error {
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

func generateCode(n int) string {
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

type Service struct {
	store storage.Storage
}

func New(store storage.Storage) *Service {
	return &Service{store: store}
}

func (s *Service) Shorten(ctx context.Context, rawURL string) (string, error) {
	if err := validateURL(rawURL); err != nil {
		return "", err
	}

	for range maxSaveRetries {
		code := generateCode(codeLength)

		err := s.store.Save(ctx, code, rawURL)
		if err == nil {
			return code, nil
		}
		if !errors.Is(err, storage.ErrAlreadyExists) {
			return "", fmt.Errorf("save link: %w", err)
		}
		// коллизия: пробуем со следующим кодом
	}

	return "", ErrCodeGeneration
}

func (s *Service) Resolve(ctx context.Context, code string) (string, error) {
	link, err := s.store.Get(ctx, code)
	if err != nil {
		return "", fmt.Errorf("get link: %w", err)
	}

	if err := s.store.IncrementClicks(ctx, code); err != nil {
		slog.Error("increment clicks failed", "code", code, "err", err)
	}

	return link.URL, nil
}
