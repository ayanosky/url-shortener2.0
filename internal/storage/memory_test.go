package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestMemory_SaveAndGet(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	if err := m.Save(ctx, "abc", "https://go.dev"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := m.Get(ctx, "abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.URL != "https://go.dev" {
		t.Errorf("URL = %q, want %q", got.URL, "https://go.dev")
	}
}

func TestMemory_SaveDuplicate(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	if err := m.Save(ctx, "abc", "https://go.dev"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	err := m.Save(ctx, "abc", "https://go.dev")
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("err = %v, want ErrAlreadyExists", err)
	}
}

func TestMemory_GetNotFound(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	_, err := m.Get(ctx, "cr7")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMemory_IncrementClicks(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()

	if err := m.Save(ctx, "abc", "https://go.dev"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := m.IncrementClicks(ctx, "abc"); err != nil {
		t.Fatalf("IncrementClicks: %v", err)
	}

	if err := m.IncrementClicks(ctx, "abc"); err != nil {
		t.Fatalf("IncrementClicks: %v", err)
	}

	link, err := m.Get(ctx, "abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if link.Clicks != 2 {
		t.Errorf("Clicks = %d, want 2", link.Clicks)
	}

	err = m.IncrementClicks(ctx, "999aaa")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMemory_ConcurrentIncrement(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.Save(ctx, "abc", "https://go.dev")

	const n = 100
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.IncrementClicks(ctx, "abc")
		}()
	}
	wg.Wait()

	link, err := m.Get(ctx, "abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if link.Clicks != n {
		t.Errorf("Clicks = %d, want %v", link.Clicks, n)
	}
}
