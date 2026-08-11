// Package config loads and validates server configuration from the environment.
//
// The server must never bind to a non-loopback interface by default; this is
// enforced here so an accidental HOST=0.0.0.0 cannot silently expose the
// controller publicly (PRD §19).
package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultHost = "127.0.0.1"
	// defaultPort is 2014, the year Amazon Echo devices launched.
	defaultPort = 2014
)

// Config holds all runtime configuration for the server.
type Config struct {
	Host string
	Port int
	// Level controls the verbosity of structured logging.
	Level slog.Level

	// SearchEngine selects the web search backend ("google" or "duckduckgo").
	SearchEngine string
	// SearchRoots restricts Spotlight file search to these directories. An
	// empty list disables file search (PRD §9).
	SearchRoots []string

	// SkillID is the Alexa application ID of this skill; requests from any
	// other skill are rejected. Empty disables the check (dev only).
	SkillID string
	// DeveloperRoot is the project folder opened by open_backend. Empty
	// disables the command (PRD §7.5).
	DeveloperRoot string
	// CodingModeShortcut is the Shortcut run by coding_mode (PRD §15).
	CodingModeShortcut string
}

// Load reads configuration from the environment, applying defaults where a
// variable is unset, and validates the result. Unknown or invalid values are
// returned as errors rather than silently accepted.
func Load() (Config, error) {
	cfg := Config{
		Host: envOr("HOST", defaultHost),
		Port: defaultPort,
	}

	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("PORT must be an integer: %q", v)
		}
		if p < 1 || p > 65535 {
			return Config{}, fmt.Errorf("PORT must be between 1 and 65535: %d", p)
		}
		cfg.Port = p
	}

	if cfg.Host == "0.0.0.0" || cfg.Host == "::" || cfg.Host == "" {
		return Config{}, fmt.Errorf("HOST %q is not allowed: the controller must bind to a loopback address", cfg.Host)
	}

	level, err := parseLevel(envOr("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	cfg.Level = level

	engine := envOr("SEARCH_ENGINE", "google")
	if engine != "google" && engine != "duckduckgo" {
		return Config{}, fmt.Errorf("SEARCH_ENGINE must be google or duckduckgo: %q", engine)
	}
	cfg.SearchEngine = engine

	cfg.SearchRoots, err = parseSearchRoots(os.Getenv("SEARCH_ROOTS"))
	if err != nil {
		return Config{}, err
	}

	cfg.SkillID = os.Getenv("ALEXA_SKILL_ID")

	cfg.DeveloperRoot, err = expandHome(os.Getenv("DEVELOPER_ROOT"))
	if err != nil {
		return Config{}, fmt.Errorf("DEVELOPER_ROOT: %w", err)
	}
	cfg.CodingModeShortcut = os.Getenv("SHORTCUT_CODING_MODE")

	return cfg, nil
}

// Addr returns the host:port the server should listen on.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLevel(s string) (slog.Level, error) {
	switch s {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error: %q", s)
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) (string, error) {
	if path == "" || (!strings.HasPrefix(path, "~")) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot expand %q: %w", path, err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

// parseSearchRoots splits a comma-separated directory list and expands a
// leading ~ to the user's home directory.
func parseSearchRoots(v string) ([]string, error) {
	if v == "" {
		return nil, nil
	}
	var roots []string
	for _, part := range strings.Split(v, ",") {
		root := strings.TrimSpace(part)
		if root == "" {
			continue
		}
		expanded, err := expandHome(root)
		if err != nil {
			return nil, err
		}
		roots = append(roots, expanded)
	}
	return roots, nil
}
