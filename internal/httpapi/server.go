// Package httpapi exposes the controller's HTTP surface.
//
// Every route is wrapped in the same middleware chain: request IDs,
// structured request logging, panic recovery, security headers and a body
// size limit. Handlers only ever return JSON.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// VersionInfo is returned by GET /version.
type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Options configures the API surface.
type Options struct {
	Logger  *slog.Logger
	Version string
	Commit  string
	Date    string
}

// New builds the full HTTP handler: routes plus middleware.
func New(opts Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /version", handleVersion(opts))

	return WithMiddleware(opts.Logger, mux)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleVersion(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, VersionInfo{
			Version: opts.Version,
			Commit:  opts.Commit,
			Date:    opts.Date,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
