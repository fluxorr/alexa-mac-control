// Package commands defines the allowlist abstraction at the heart of the
// system (PRD §5, §6). Every Mac-affecting action is a named Command in the
// registry; nothing else may execute anything. Neither the Alexa layer nor
// the HTTP layer ever runs a shell directly.
package commands

import "context"

// CommandArgs holds validated, bounded arguments for a command.
type CommandArgs map[string]string

// Result is the outcome of a command execution.
type Result struct {
	// Message is a human-friendly summary of what happened.
	Message string `json:"message"`
	// Data carries optional structured payloads (system status, search hits).
	Data any `json:"data,omitempty"`
}

// Command is a single allowlisted action.
type Command struct {
	Name        string
	Description string
	Execute     func(ctx context.Context, args CommandArgs) (Result, error)
}
