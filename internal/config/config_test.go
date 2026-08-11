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
	if cfg.Port != 2014 {
		t.Errorf("Port = %d, want 2014", cfg.Port)
	}
	if cfg.Level != slog.LevelInfo {
		t.Errorf("Level = %v, want info", cfg.Level)
	}
	if got := cfg.Addr(); got != "127.0.0.1:2014" {
		t.Errorf("Addr() = %q, want 127.0.0.1:2014", got)
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

func TestLoadSearchDefaults(t *testing.T) {
	t.Setenv("SEARCH_ENGINE", "")
	t.Setenv("SEARCH_ROOTS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SearchEngine != "google" {
		t.Errorf("SearchEngine = %q, want google", cfg.SearchEngine)
	}
	if len(cfg.SearchRoots) != 0 {
		t.Errorf("SearchRoots = %v, want none", cfg.SearchRoots)
	}
}

func TestLoadRejectsInvalidSearchEngine(t *testing.T) {
	t.Setenv("SEARCH_ENGINE", "bing")
	t.Setenv("SEARCH_ROOTS", "")
	if _, err := Load(); err == nil {
		t.Error("Load() with SEARCH_ENGINE=bing: want error, got nil")
	}
}

func TestLoadSearchRoots(t *testing.T) {
	t.Setenv("SEARCH_ENGINE", "duckduckgo")
	t.Setenv("SEARCH_ROOTS", "~/Developer, ~/Documents ,,")
	t.Setenv("HOME", "/Users/test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SearchEngine != "duckduckgo" {
		t.Errorf("SearchEngine = %q, want duckduckgo", cfg.SearchEngine)
	}
	want := []string{"/Users/test/Developer", "/Users/test/Documents"}
	if len(cfg.SearchRoots) != len(want) {
		t.Fatalf("SearchRoots = %v, want %v", cfg.SearchRoots, want)
	}
	for i := range want {
		if cfg.SearchRoots[i] != want[i] {
			t.Errorf("SearchRoots[%d] = %q, want %q", i, cfg.SearchRoots[i], want[i])
		}
	}
}

func TestParseSearchRootsTildeOnly(t *testing.T) {
	t.Setenv("HOME", "/Users/test")
	roots, err := parseSearchRoots("~")
	if err != nil {
		t.Fatalf("parseSearchRoots error = %v", err)
	}
	if len(roots) != 1 || roots[0] != "/Users/test" {
		t.Errorf("parseSearchRoots(~) = %v, want [/Users/test]", roots)
	}
}
