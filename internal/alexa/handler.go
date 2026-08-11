package alexa

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/fluxorr/alexa-mac-control/internal/commands"
)

// Handler is the public Alexa endpoint: it verifies every request, then
// resolves intents to commands. Only this handler may be exposed through
// Cloudflare; /api/command never is (PRD §20, §24).
type Handler struct {
	logger   *slog.Logger
	verifier *Verifier
	commands *commands.Registry
}

// New builds the Alexa handler. A nil verifier disables verification, which
// is only acceptable for local development with no public exposure.
func New(logger *slog.Logger, v *Verifier, reg *commands.Registry) *Handler {
	return &Handler{logger: logger, verifier: v, commands: reg}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	if h.verifier != nil {
		if err := h.verifier.Verify(r.Context(), raw, r.Header); err != nil {
			h.logger.Warn("alexa request rejected", "error", err.Error())
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if h.verifier != nil {
		if err := h.verifier.CheckSkillID(&req); err != nil {
			h.logger.Warn("alexa skill ID mismatch", "error", err.Error())
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	h.logger.Info("alexa request",
		"type", req.Request.Type,
		"intent", req.Request.Intent.Name,
	)

	resp := h.Handle(r.Context(), &req)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("encode alexa response", "error", err)
	}
}

// ErrUnauthorized is returned by the verifier when a request fails any
// verification step.
var ErrUnauthorized = errors.New("unauthorized alexa request")
