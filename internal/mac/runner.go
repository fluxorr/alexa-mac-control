// Package mac implements the macOS side of the controller (PRD §35). It
// knows nothing about Alexa or the HTTP layer: it exposes plain functions
// that execute fixed, allowlisted actions through a Runner.
//
// Every external program is invoked with explicit argv elements — no shell
// is involved anywhere in this package.
package mac

import (
	"context"
	"os/exec"
)

// Runner executes external programs. The production implementation shells
// out via os/exec with no shell involved; tests inject a fake that records
// calls and returns canned output, so the test suite can never touch the
// real system (PRD §25).
type Runner interface {
	// Run executes a program to completion.
	Run(ctx context.Context, name string, args ...string) error
	// Output executes a program and returns its combined output.
	Output(ctx context.Context, name string, args ...string) (string, error)
}

// NewRunner returns the real OS-backed Runner.
func NewRunner() Runner {
	return execRunner{}
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func (execRunner) Output(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}
