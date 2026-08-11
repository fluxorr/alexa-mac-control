package commands

import (
	"context"
	"errors"
	"testing"
)

func TestRegisterAndLookup(t *testing.T) {
	reg := NewRegistry()
	cmd := &Command{Name: "ping", Execute: func(context.Context, CommandArgs) (Result, error) {
		return Result{Message: "pong"}, nil
	}}
	reg.Register(cmd)

	got, ok := reg.Lookup("ping")
	if !ok {
		t.Fatal("Lookup(ping): want ok")
	}
	if got != cmd {
		t.Errorf("Lookup(ping) = %v, want registered command", got)
	}

	if _, ok := reg.Lookup("nope"); ok {
		t.Error("Lookup(nope): want not ok")
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	reg := NewRegistry()
	if _, err := reg.Execute(context.Background(), "rm_rf", nil); err == nil {
		t.Error("Execute(unknown): want error, got nil")
	}
}

func TestExecuteRunsCommand(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Command{Name: "ping", Execute: func(ctx context.Context, args CommandArgs) (Result, error) {
		if got := args["query"]; got != "hello" {
			t.Errorf("query arg = %q, want hello", got)
		}
		return Result{Message: "pong"}, nil
	}})

	res, err := reg.Execute(context.Background(), "ping", CommandArgs{"query": "hello"})
	if err != nil {
		t.Fatalf("Execute(ping) error = %v", err)
	}
	if res.Message != "pong" {
		t.Errorf("Message = %q, want pong", res.Message)
	}
}

func TestExecutePropagatesCommandError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Command{Name: "boom", Execute: func(context.Context, CommandArgs) (Result, error) {
		return Result{}, errors.New("failed")
	}})

	if _, err := reg.Execute(context.Background(), "boom", nil); err == nil {
		t.Error("Execute(boom): want error, got nil")
	}
}

func TestNamesSorted(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Command{Name: "zeta"})
	reg.Register(&Command{Name: "alpha"})
	reg.Register(&Command{Name: "mid"})

	got := reg.Names()
	want := []string{"alpha", "mid", "zeta"}
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
