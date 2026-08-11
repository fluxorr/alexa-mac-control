package config

import (
	"log/slog"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", cfg.Host)
	}
	if cfg.Port != 8787 {
		t.Errorf("Port = %d, want 8787", cfg.Port)
	}
	if cfg.Level != slog.LevelInfo {
		t.Errorf("Level = %v, want info", cfg.Level)
	}
	if got := cfg.Addr(); got != "127.0.0.1:8787" {
		t.Errorf("Addr() = %q, want 127.0.0.1:8787", got)
	}
}

func TestLoadCustom(t *testing.T) {
	t.Setenv("HOST", "127.0.0.2")
	t.Setenv("PORT", "9000")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Host != "127.0.0.2" || cfg.Port != 9000 || cfg.Level != slog.LevelDebug {
		t.Errorf("Load() = %+v, want custom values", cfg)
	}
}

func TestLoadRejectsNonLoopbackHost(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "::"} {
		t.Setenv("HOST", host)
		t.Setenv("PORT", "")
		t.Setenv("LOG_LEVEL", "")
		if _, err := Load(); err == nil {
			t.Errorf("Load() with HOST=%q: want error, got nil", host)
		}
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	for _, port := range []string{"abc", "0", "70000", "-1"} {
		t.Setenv("PORT", port)
		t.Setenv("HOST", "")
		t.Setenv("LOG_LEVEL", "")
		if _, err := Load(); err == nil {
			t.Errorf("Load() with PORT=%q: want error, got nil", port)
		}
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := Load(); err == nil {
		t.Error("Load() with LOG_LEVEL=verbose: want error, got nil")
	}
}
