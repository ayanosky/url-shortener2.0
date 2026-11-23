package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("missing database url", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")

		if _, err := Load(); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("defaults", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("PORT", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Port != "8080" {
			t.Errorf("Port = %q, want %q", cfg.Port, "8080")
		}
	})

	t.Run("custom port", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("PORT", "9090")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Port != "9090" {
			t.Errorf("Port = %q, want %q", cfg.Port, "9090")
		}
	})
}
