package service

import (
	"context"

	"github.com/ayanosky/url-shortener2.0/internal/storage"
)

type Service struct {
	store storage.Storage
}

func New(store storage.Storage) *Service {
	return &Service{store: store}
}

func (s *Service) Shorten(ctx context.Context, rawURL string) (string, error) {
	// 1. валидация URL
	// 2. генерация кода
	// 3. store.Save; если ErrAlreadyExists, сгенерировать новый код и повторить (до N раз)
	return "", nil
}
