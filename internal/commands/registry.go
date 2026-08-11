package commands

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Registry is the allowlist of executable commands. Unknown names are never
// resolved: callers must check the ok result.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*Command
}

func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]*Command)}
}

func (r *Registry) Register(cmd *Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.Name] = cmd
}

func (r *Registry) Lookup(name string) (*Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.commands[name]
	return cmd, ok
}

// Names returns the sorted command names, for logging and help output.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Execute runs the named command, rejecting anything outside the allowlist.
func (r *Registry) Execute(ctx context.Context, name string, args CommandArgs) (Result, error) {
	cmd, ok := r.Lookup(name)
	if !ok {
		return Result{}, fmt.Errorf("unknown command %q", name)
	}
	return cmd.Execute(ctx, args)
}
