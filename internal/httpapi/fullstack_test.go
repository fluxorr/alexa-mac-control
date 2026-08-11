package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fluxorr/alexa-mac-control/internal/commands"
	"github.com/fluxorr/alexa-mac-control/internal/mac"
)

// stackRunner records mac calls end to end, standing in for the OS.
type stackRunner struct {
	calls []string
}

func (s *stackRunner) Run(_ context.Context, name string, args ...string) error {
	s.calls = append(s.calls, name+" "+strings.Join(args, " "))
	return nil
}

func (s *stackRunner) Output(_ context.Context, name string, args ...string) (string, error) {
	s.calls = append(s.calls, name+" "+strings.Join(args, " "))
	return "", nil
}

// TestFullStack exercises the complete local path: POST /api/command through
// the registry and the real built-in command wiring down to the Runner
// abstraction (PRD §31: MVP local flow).
func TestFullStack(t *testing.T) {
	runner := &stackRunner{}
	reg := commands.NewRegistry()
	commands.RegisterDefaults(reg, commands.Defaults{
		Runner:       runner,
		SearchEngine: mac.EngineDuckDuckGo,
		SearchRoots:  []string{"/Users/me/Developer"},
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(Options{Logger: logger, Commands: reg})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/command",
		strings.NewReader(`{"command":"search_web","query":"go closures"}`)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var res commands.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if res.Message != "Searching for go closures." {
		t.Errorf("Message = %q, want Searching for go closures.", res.Message)
	}
	wantCall := "open https://duckduckgo.com/?q=go+closures"
	if len(runner.calls) == 0 || runner.calls[len(runner.calls)-1] != wantCall {
		t.Errorf("last call = %v, want %q", runner.calls, wantCall)
	}
}
