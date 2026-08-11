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
	"strconv"
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
