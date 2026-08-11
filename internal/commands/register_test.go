package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxorr/alexa-mac-control/internal/mac"
)

// stubRunner is a minimal mac.Runner fake for the command layer: it records
// calls and serves canned output. Tests never touch the real system.
type stubRunner struct {
	calls []string
	out   map[string]string
}

func (s *stubRunner) key(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

func (s *stubRunner) Run(_ context.Context, name string, args ...string) error {
	s.calls = append(s.calls, s.key(name, args))
	return nil
}

func (s *stubRunner) Output(_ context.Context, name string, args ...string) (string, error) {
	key := s.key(name, args)
	s.calls = append(s.calls, key)
	return s.out[key], nil
}

func testDefaults(stub *stubRunner) Defaults {
	return Defaults{
		Runner:       stub,
		SearchEngine: mac.EngineGoogle,
		SearchRoots:  []string{"/Users/me/Developer"},
	}
}

func newTestRegistry(t *testing.T) (*Registry, *stubRunner) {
	t.Helper()
	stub := &stubRunner{out: map[string]string{}}
	reg := NewRegistry()
	RegisterDefaults(reg, testDefaults(stub))
	return reg, stub
}

func TestRegisterDefaults(t *testing.T) {
	reg, _ := newTestRegistry(t)
	want := []string{
		"open_spotify", "open_vscode", "open_terminal", "open_safari",
		"search_web", "search_files", "system_status",
	}
	for _, name := range want {
		if _, ok := reg.Lookup(name); !ok {
			t.Errorf("command %q not registered", name)
		}
	}
}

func TestOpenAppCommands(t *testing.T) {
	reg, stub := newTestRegistry(t)
	stub.out["mdfind kMDItemCFBundleIdentifier == 'com.spotify.client'"] = "/Applications/Spotify.app\n"

	res, err := reg.Execute(context.Background(), "open_spotify", nil)
	if err != nil {
		t.Fatalf("open_spotify error = %v", err)
	}
	if res.Message != "Opening Spotify." {
		t.Errorf("Message = %q, want Opening Spotify.", res.Message)
	}
	if len(stub.calls) == 0 || !strings.Contains(stub.calls[len(stub.calls)-1], "open /Applications/Spotify.app") {
		t.Errorf("open was not called; calls = %v", stub.calls)
	}
}

func TestSearchWeb(t *testing.T) {
	reg, stub := newTestRegistry(t)

	res, err := reg.Execute(context.Background(), "search_web", CommandArgs{"query": "go closures"})
	if err != nil {
		t.Fatalf("search_web error = %v", err)
	}
	if res.Message != "Searching for go closures." {
		t.Errorf("Message = %q, want Searching for go closures.", res.Message)
	}
	if !strings.Contains(stub.calls[len(stub.calls)-1], "open https://www.google.com/search?q=go+closures") {
		t.Errorf("unexpected open call; calls = %v", stub.calls)
	}
}

func TestSearchWebMissingQuery(t *testing.T) {
	reg, stub := newTestRegistry(t)
	if _, err := reg.Execute(context.Background(), "search_web", nil); err == nil {
		t.Error("search_web without query: want error")
	}
	if len(stub.calls) != 0 {
		t.Errorf("no mac call expected; calls = %v", stub.calls)
	}
}

func TestSearchFilesMessage(t *testing.T) {
	reg, stub := newTestRegistry(t)
	stub.out["mdfind -onlyin /Users/me/Developer auth middleware"] =
		"/Users/me/Developer/handler.go\n/Users/me/Developer/auth.go\n/Users/me/Developer/middleware.go\n/Users/me/Developer/main.go\n"

	res, err := reg.Execute(context.Background(), "search_files", CommandArgs{"query": "auth middleware"})
	if err != nil {
		t.Fatalf("search_files error = %v", err)
	}
	want := "I found 4 matching files. The first 3 are handler.go, auth.go, and middleware.go."
	if res.Message != want {
		t.Errorf("Message = %q, want %q", res.Message, want)
	}
}

func TestSearchFilesOneHit(t *testing.T) {
	reg, stub := newTestRegistry(t)
	stub.out["mdfind -onlyin /Users/me/Developer config"] = "/Users/me/Developer/config.yaml\n"

	res, err := reg.Execute(context.Background(), "search_files", CommandArgs{"query": "config"})
	if err != nil {
		t.Fatalf("search_files error = %v", err)
	}
	if res.Message != "I found one file: config.yaml." {
		t.Errorf("Message = %q, want one-file message", res.Message)
	}
}

func TestSearchFilesNoHits(t *testing.T) {
	reg, stub := newTestRegistry(t)
	stub.out["mdfind -onlyin /Users/me/Developer nothing"] = ""

	res, err := reg.Execute(context.Background(), "search_files", CommandArgs{"query": "nothing"})
	if err != nil {
		t.Fatalf("search_files error = %v", err)
	}
	if res.Message != "I couldn't find anything matching that." {
		t.Errorf("Message = %q, want no-hits message", res.Message)
	}
}

func TestSearchFilesNotConfigured(t *testing.T) {
	stub := &stubRunner{out: map[string]string{}}
	reg := NewRegistry()
	RegisterDefaults(reg, Defaults{Runner: stub, SearchEngine: mac.EngineGoogle})

	res, err := reg.Execute(context.Background(), "search_files", CommandArgs{"query": "x"})
	if err != nil {
		t.Fatalf("search_files error = %v", err)
	}
	if res.Message != "File search is not configured on this Mac." {
		t.Errorf("Message = %q, want not-configured message", res.Message)
	}
	if len(stub.calls) != 0 {
		t.Errorf("no mac call expected without roots; calls = %v", stub.calls)
	}
}

func TestSystemStatusMessage(t *testing.T) {
	reg, stub := newTestRegistry(t)
	stub.out["sysctl -n kern.boottime"] = "{ sec = 1752345600 } 123456789 0\n"
	stub.out["top -l 1 -n 0"] = "CPU usage: 10.0% user, 8.0% sys, 82.0% idle\n"
	stub.out["sysctl -n hw.memsize"] = "17179869184\n"
	stub.out["vm_stat"] = "Mach Virtual Memory Statistics: (page size of 4096 bytes)\nPages active: 1500000.\nPages wired down: 200000.\nPages occupied by compressor: 150000.\n"
	stub.out["pmset -g batt"] = "-InternalBattery-0 (id=1)\t74%; discharging; 4:02 remaining present: true\n"
	stub.out["df -k /"] = "/dev/disk3s1 245110784 135202900 108834076    56% 1 2 3   /\n"

	res, err := reg.Execute(context.Background(), "system_status", nil)
	if err != nil {
		t.Fatalf("system_status error = %v", err)
	}
	want := "Your Mac is online. CPU usage is 18 percent, memory usage is 44 percent, and battery is at 74 percent."
	if res.Message != want {
		t.Errorf("Message = %q, want %q", res.Message, want)
	}
}

func TestJoinNames(t *testing.T) {
	for _, tt := range []struct {
		in   []string
		want string
	}{
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a and b"},
		{[]string{"a", "b", "c"}, "a, b, and c"},
	} {
		if got := joinNames(tt.in); got != tt.want {
			t.Errorf("joinNames(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
