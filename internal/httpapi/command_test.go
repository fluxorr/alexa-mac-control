package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fluxorr/alexa-mac-control/internal/commands"
	"github.com/fluxorr/alexa-mac-control/internal/security"
)

func newTestRegistry() *commands.Registry {
	reg := commands.NewRegistry()
	reg.Register(&commands.Command{
		Name:        "ping",
		Description: "test command",
		Execute: func(ctx context.Context, args commands.CommandArgs) (commands.Result, error) {
			return commands.Result{Message: "pong"}, nil
		},
	})
	reg.Register(&commands.Command{
		Name: "boom",
		Execute: func(context.Context, commands.CommandArgs) (commands.Result, error) {
			return commands.Result{}, errors.New("execution failed")
		},
	})
	return reg
}

func testCommandHandler(t *testing.T) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(Options{Logger: logger, Version: "test", Commands: newTestRegistry()})
}

func postCommand(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/command", strings.NewReader(body)))
	return rec
}

func TestCommandOK(t *testing.T) {
	rec := postCommand(t, testCommandHandler(t), `{"command":"ping"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var res commands.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if res.Message != "pong" {
		t.Errorf("Message = %q, want pong", res.Message)
	}
}

func TestCommandWithQuery(t *testing.T) {
	h := testCommandHandler(t)
	rec := postCommand(t, h, `{"command":"ping","query":"go closures"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCommandUnknown(t *testing.T) {
	rec := postCommand(t, testCommandHandler(t), `{"command":"rm_rf"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestCommandMissingName(t *testing.T) {
	rec := postCommand(t, testCommandHandler(t), `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCommandInvalidJSON(t *testing.T) {
	rec := postCommand(t, testCommandHandler(t), `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCommandUnknownFieldRejected(t *testing.T) {
	rec := postCommand(t, testCommandHandler(t), `{"command":"ping","evil":"true"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCommandQueryTooLong(t *testing.T) {
	rec := postCommand(t, testCommandHandler(t),
		`{"command":"ping","query":"`+strings.Repeat("a", security.MaxQueryLength+1)+`"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCommandExecutionFailure(t *testing.T) {
	rec := postCommand(t, testCommandHandler(t), `{"command":"boom"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	// Internal details must not leak to the client.
	if strings.Contains(rec.Body.String(), "execution failed") {
		t.Errorf("response leaks internal error: %s", rec.Body.String())
	}
}

func TestCommandMethodNotAllowed(t *testing.T) {
	h := testCommandHandler(t)
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/api/command", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/command: status = %d, want 405", method, rec.Code)
		}
	}
}
