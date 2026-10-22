package storage

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("link not found")
	ErrAlreadyExists = errors.New("code already exists")
)

type Link struct {
	Code   string
	URL    string
	Clicks int
}

type Storage interface {
	Save(ctx context.Context, code, url string) error
	Get(ctx context.Context, code string) (Link, error)
	IncrementClicks(ctx context.Context, code string) error
}
