// Package security contains the validation and authorization layers that
// sit between the network edge and the command registry (PRD §19).
package security

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// MaxQueryLength caps free-text arguments such as search queries. Long
	// input is more likely abuse than a legitimate request.
	MaxQueryLength = 256
	// maxCommandNameLength bounds command identifiers; real names are far
	// shorter and this only rejects nonsense input early.
	maxCommandNameLength = 64
)

// ValidateCommand ensures a command name is present and well-formed.
func ValidateCommand(name string) error {
	if name == "" {
		return errors.New("command is required")
	}
	if len(name) > maxCommandNameLength {
		return errors.New("command name too long")
	}
	return nil
}

// ValidateQuery bounds a free-text argument such as a search query.
func ValidateQuery(q string) error {
	if strings.TrimSpace(q) == "" {
		return errors.New("query is required")
	}
	if len(q) > MaxQueryLength {
		return fmt.Errorf("query too long (max %d characters)", MaxQueryLength)
	}
	return nil
}
