package alexa

import (
	"context"
	"strings"
	"time"

	"github.com/fluxorr/alexa-mac-control/internal/commands"
)

// actionCommands maps the Action slot of MacCommandIntent to allowlisted
// command names. Matching is case- and spacing-insensitive.
var actionCommands = map[string]string{
	"open spotify":            "open_spotify",
	"open vs code":            "open_vscode",
	"open visual studio code": "open_vscode",
	"open terminal":           "open_terminal",
	"open safari":             "open_safari",
	"open backend":            "open_backend",
	"open my project":         "open_backend",
	"lock":                    "lock",
	"lock my mac":             "lock",
	"sleep":                   "sleep",
	"sleep my mac":            "sleep",
	"coding mode":             "coding_mode",
	"start coding mode":       "coding_mode",
}

const intentTimeout = 30 * time.Second

// Handle resolves a verified Alexa request into a speech response. It never
// executes anything itself; it only resolves command names through the
// registry (PRD §6).
func (h *Handler) Handle(ctx context.Context, req *Request) Response {
	switch req.Request.Type {
	case RequestTypeLaunch:
		return Tell("Hi, I can control your Mac. Try asking me to open an app, search for something, or check the system status.")
	case RequestTypeSessionEnd:
		return EndSession()
	case RequestTypeIntent:
		return h.handleIntent(ctx, &req.Request)
	default:
		h.logger.Warn("unknown request type", "type", req.Request.Type)
		return Tell("I didn't understand that request.")
	}
}

func (h *Handler) handleIntent(ctx context.Context, r *IntentRequest) Response {
	switch r.Intent.Name {
	case IntentMacStatus:
		return h.run(ctx, "system_status", nil)
	case IntentMacSearch:
		query := slotValue(r, "Query")
		if query == "" {
			return Tell("What would you like me to search for?")
		}
		return h.run(ctx, "search_web", commands.CommandArgs{"query": query})
	case IntentMacFileSearch:
		query := slotValue(r, "Query")
		if query == "" {
			return Tell("What would you like me to search your files for?")
		}
		return h.run(ctx, "search_files", commands.CommandArgs{"query": query})
	case IntentMacCommand:
		return h.handleAction(ctx, slotValue(r, "Action"))
	case IntentHelp:
		return Tell("I can open apps like Spotify and Visual Studio Code, lock or put your Mac to sleep, search the web, search your files, and report your system status. Just ask.")
	case IntentCancel, IntentStop:
		return Tell("Goodbye.")
	case IntentFallback:
		return Tell("I didn't catch that. You can ask me to open an app, search, lock, sleep, or check the system status.")
	default:
		h.logger.Warn("unknown intent", "intent", r.Intent.Name)
		return Tell("I didn't understand that request.")
	}
}

func (h *Handler) handleAction(ctx context.Context, action string) Response {
	name, ok := actionCommands[normalizeAction(action)]
	if !ok {
		return Tell("I can't do that yet. I can open Spotify, Visual Studio Code, Terminal, or Safari, open your backend project, start coding mode, lock, sleep, search, or check the system status.")
	}
	return h.run(ctx, name, nil)
}

// run executes an allowlisted command and renders its friendly message.
func (h *Handler) run(ctx context.Context, name string, args commands.CommandArgs) Response {
	ctx, cancel := context.WithTimeout(ctx, intentTimeout)
	defer cancel()

	result, err := h.commands.Execute(ctx, name, args)
	if err != nil {
		h.logger.Error("command execution failed", "command", name, "error", err)
		return Tell("Sorry, I couldn't do that.")
	}
	return Tell(result.Message)
}

func slotValue(r *IntentRequest, name string) string {
	slot, ok := r.Intent.Slots[name]
	if !ok {
		return ""
	}
	return strings.TrimSpace(slot.Value)
}

// normalizeAction lowercases and collapses whitespace so "Open   VS Code"
// matches "open vs code".
func normalizeAction(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
