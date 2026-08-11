package mac

import (
	"context"
	"strings"
	"sync"
	"testing"
)

type call struct {
	name string
	args []string
}

// fakeRunner records every invocation and serves canned output or errors
// keyed by the full command line. It exists so tests exercise the real
// parsing and command construction without touching the system (PRD §25).
type fakeRunner struct {
	mu      sync.Mutex
	calls   []call
	outputs map[string]string
	errs    map[string]error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{}, errs: map[string]error{}}
}

func (f *fakeRunner) withOutput(cmd, out string) *fakeRunner {
	f.outputs[cmd] = out
	return f
}

func (f *fakeRunner) withError(cmd string, err error) *fakeRunner {
	f.errs[cmd] = err
	return f
}

func (f *fakeRunner) key(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) error {
	f.mu.Lock()
	f.calls = append(f.calls, call{name, args})
	err := f.errs[f.key(name, args)]
	f.mu.Unlock()
	return err
}

func (f *fakeRunner) Output(ctx context.Context, name string, args ...string) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, call{name, args})
	out, err := f.outputs[f.key(name, args)], f.errs[f.key(name, args)]
	f.mu.Unlock()
	return out, err
}

func (f *fakeRunner) ran(name string, args ...string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if c.name == name && slicesEqual(c.args, args) {
			return true
		}
	}
	return false
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestOpenApp(t *testing.T) {
	fake := newFakeRunner().withOutput(
		"mdfind kMDItemCFBundleIdentifier == 'com.spotify.client'",
		"/Applications/Spotify.app\n/Applications/Spotify2.app\n",
	)

	if err := OpenApp(context.Background(), fake, BundleSpotify); err != nil {
		t.Fatalf("OpenApp error = %v", err)
	}
	if !fake.ran("open", "/Applications/Spotify.app") {
		t.Errorf("open was not called with the detected path; calls = %v", fake.calls)
	}
}

func TestOpenAppNotInstalled(t *testing.T) {
	fake := newFakeRunner().withOutput(
		"mdfind kMDItemCFBundleIdentifier == 'com.spotify.client'",
		"",
	)
	if err := OpenApp(context.Background(), fake, BundleSpotify); err == nil {
		t.Error("OpenApp: want error when app is not installed, got nil")
	}
	if fake.ran("open") {
		t.Error("open must not be called when the app is missing")
	}
}

func TestRunShortcut(t *testing.T) {
	fake := newFakeRunner()
	if err := RunShortcut(context.Background(), fake, "Mac - Coding Mode"); err != nil {
		t.Fatalf("RunShortcut error = %v", err)
	}
	if !fake.ran("shortcuts", "run", "Mac - Coding Mode") {
		t.Errorf("shortcuts run was not called correctly; calls = %v", fake.calls)
	}
}
