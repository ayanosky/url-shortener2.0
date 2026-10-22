package storage

import (
	"context"
	"sync"
)

// Проверка, что интерфейс реализован
var _ Storage = (*Memory)(nil)

type Memory struct {
	mu    sync.RWMutex
	links map[string]*Link
}

func NewMemory() *Memory {
	return &Memory{links: make(map[string]*Link)}
}

func (m *Memory) Save(ctx context.Context, code, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.links[code]; ok {
		return ErrAlreadyExists
	}
	m.links[code] = &Link{Code: code, URL: url}
	return nil
}

func (m *Memory) Get(ctx context.Context, code string) (Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	link, ok := m.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}
	return *link, nil
}

func (m *Memory) IncrementClicks(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	link, ok := m.links[code]
	if !ok {
		return ErrNotFound
	}
	link.Clicks++
	return nil
}
