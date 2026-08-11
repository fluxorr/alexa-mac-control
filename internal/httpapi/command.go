package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/fluxorr/alexa-mac-control/internal/commands"
	"github.com/fluxorr/alexa-mac-control/internal/security"
)

// commandTimeout bounds every command execution. A command that cannot
// finish in this window is hung and must not tie up the server.
const commandTimeout = 30 * time.Second

// commandRequest is the body of POST /api/command. This endpoint is for
// local development only (PRD §24); the public Alexa path never reaches it.
type commandRequest struct {
	Command string `json:"command"`
	Query   string `json:"query,omitempty"`
}

func handleCommand(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opts.Commands == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "command registry not configured"})
			return
		}

		var req commandRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		if err := security.ValidateCommand(req.Command); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, ok := opts.Commands.Lookup(req.Command); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown command"})
			return
		}
		if req.Query != "" {
			if err := security.ValidateQuery(req.Query); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), commandTimeout)
		defer cancel()

		result, err := opts.Commands.Execute(ctx, req.Command, commands.CommandArgs{"query": req.Query})
		if err != nil {
			opts.Logger.Error("command execution failed",
				"command", req.Command,
				"error", err,
			)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "command failed"})
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}
