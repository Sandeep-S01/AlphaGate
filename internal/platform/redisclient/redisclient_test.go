package redisclient

import (
	"testing"

	"sentra/internal/config"
)

func TestOptionsFromConfigUsesConnectionSettings(t *testing.T) {
	cfg := config.RedisConfig{
		Addr:     "redis:6379",
		Password: "secret",
		DB:       3,
	}

	options := OptionsFromConfig(cfg)

	if options.Addr != "redis:6379" {
		t.Fatalf("expected addr redis:6379, got %q", options.Addr)
	}
	if options.Password != "secret" {
		t.Fatal("expected password to be copied")
	}
	if options.DB != 3 {
		t.Fatalf("expected DB 3, got %d", options.DB)
	}
}
