package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	// maxBodyBytes caps request bodies. Alexa payloads are small; anything
	// larger is a sign of abuse and rejected outright.
	maxBodyBytes = 1 << 20 // 1 MiB

	requestIDHeader = "X-Request-ID"
)

// WithMiddleware wraps h in the shared middleware chain, innermost first:
// body limit → request ID → recovery → logging → security headers.
func WithMiddleware(logger *slog.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set(requestIDHeader, requestID)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"panic", err,
				)
				rec.Header().Set("Content-Type", "application/json")
				if rec.wroteHeader {
					return
				}
				rec.WriteHeader(http.StatusInternalServerError)
				_, _ = rec.Write([]byte(`{"error":"internal server error"}`))
			}
			logRequest(logger, requestID, r, rec)
		}()

		start := time.Now()
		h.ServeHTTP(rec, r)

		setSecurityHeaders(rec.Header())
		rec.mu.Lock()
		rec.duration = time.Since(start)
		rec.mu.Unlock()
	})
}

func logRequest(logger *slog.Logger, requestID string, r *http.Request, rec *statusRecorder) {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	args := []any{
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"remote", r.RemoteAddr,
		"status", rec.status,
		"bytes", rec.bytes,
		"duration_ms", rec.duration.Milliseconds(),
	}
	if rec.status >= 500 {
		logger.Error("request failed", args...)
		return
	}
	logger.Info("request", args...)
}

func setSecurityHeaders(h http.Header) {
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-store")
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand only fails on catastrophic system failure; a time-based
		// fallback keeps the server alive rather than refusing requests.
		return time.Now().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(b[:])
}

// statusRecorder captures the status and size of a response so middleware can
// log and react to it after the handler has run.
type statusRecorder struct {
	http.ResponseWriter
	mu          sync.Mutex
	status      int
	bytes       int
	wroteHeader bool
	duration    time.Duration
}

func (r *statusRecorder) WriteHeader(status int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.wroteHeader {
		r.status = http.StatusOK
		r.wroteHeader = true
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}
