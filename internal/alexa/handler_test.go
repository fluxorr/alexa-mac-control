package alexa

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fluxorr/alexa-mac-control/internal/commands"
	"github.com/fluxorr/alexa-mac-control/internal/mac"
)

// alexaStub is a mac.Runner fake so intent tests can use the real built-in
// command wiring without touching the system.
type alexaStub struct {
	outputs map[string]string
}

func (a alexaStub) key(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

func (a alexaStub) Run(context.Context, string, ...string) error { return nil }
func (a alexaStub) Output(_ context.Context, name string, args ...string) (string, error) {
	return a.outputs[a.key(name, args)], nil
}

// stubSystem returns a runner whose canned output satisfies every read the
// built-in commands perform.
func stubSystem() alexaStub {
	return alexaStub{outputs: map[string]string{
		"mdfind kMDItemCFBundleIdentifier == 'com.spotify.client'":    "/Applications/Spotify.app\n",
		"mdfind kMDItemCFBundleIdentifier == 'com.microsoft.VSCode'":  "/Applications/Visual Studio Code.app\n",
		"mdfind kMDItemCFBundleIdentifier == 'com.apple.Terminal'":    "/System/Applications/Utilities/Terminal.app\n",
		"mdfind kMDItemCFBundleIdentifier == 'com.apple.Safari'":      "/Applications/Safari.app\n",
		"sysctl -n kern.boottime": "{ sec = 1752345600 } 123456 0\n",
		"top -l 1 -n 0":              "CPU usage: 10.0% user, 8.0% sys, 82.0% idle\n",
		"sysctl -n hw.memsize":       "17179869184\n",
		"vm_stat":                    "Mach Virtual Memory Statistics: (page size of 4096 bytes)\nPages active: 100.\nPages wired down: 50.\nPages occupied by compressor: 25.\n",
		"pmset -g batt":              "-InternalBattery-0 (id=1)\t74%; discharging; 4:02 remaining present: true\n",
		"df -k /":                    "/dev/disk3s1 245110784 135202900 108834076    56% 1 2 3   /\n",
		"mdfind -onlyin /Users/me/Developer middleware": "",
	}}
}

func testHandler(t *testing.T) *Handler {
	t.Helper()
	reg := commands.NewRegistry()
	commands.RegisterDefaults(reg, commands.Defaults{
		Runner:       stubSystem(),
		SearchEngine: mac.EngineGoogle,
		SearchRoots:  []string{"/Users/me/Developer"},
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(logger, nil, reg) // nil verifier: verification tested separately
}

func request(t *testing.T, typeName, intent string, slots map[string]string) *Request {
	t.Helper()
	req := &Request{
		Version: "1.0",
		Request: IntentRequest{
			Type:   typeName,
			Intent: Intent{Name: intent, Slots: map[string]Slot{}},
		},
	}
	for name, value := range slots {
		req.Request.Intent.Slots[name] = Slot{Name: name, Value: value}
	}
	return req
}

func doHandle(t *testing.T, h *Handler, req *Request) Response {
	t.Helper()
	resp := h.Handle(context.Background(), req)
	if resp.Response.OutputSpeech == nil {
		t.Fatal("response has no output speech")
	}
	return resp
}

func speech(t *testing.T, resp Response) string {
	t.Helper()
	return resp.Response.OutputSpeech.Text
}

func TestIntentStatus(t *testing.T) {
	h := testHandler(t)
	got := speech(t, doHandle(t, h, request(t, RequestTypeIntent, IntentMacStatus, nil)))
	if !strings.Contains(got, "Your Mac is online") {
		t.Errorf("status speech = %q", got)
	}
}

func TestIntentSearchWeb(t *testing.T) {
	h := testHandler(t)
	got := speech(t, doHandle(t, h, request(t, RequestTypeIntent, IntentMacSearch,
		map[string]string{"Query": "go closures"})))
	if !strings.Contains(got, "Searching for go closures") {
		t.Errorf("search speech = %q", got)
	}
}

func TestIntentSearchWebMissingQuery(t *testing.T) {
	h := testHandler(t)
	got := speech(t, doHandle(t, h, request(t, RequestTypeIntent, IntentMacSearch, nil)))
	if !strings.Contains(got, "search for") {
		t.Errorf("prompt speech = %q", got)
	}
}

func TestIntentFileSearch(t *testing.T) {
	h := testHandler(t)
	got := speech(t, doHandle(t, h, request(t, RequestTypeIntent, IntentMacFileSearch,
		map[string]string{"Query": "middleware"})))
	if !strings.Contains(got, "I couldn't find anything matching that") {
		t.Errorf("file search speech = %q", got)
	}
}

func TestIntentCommandActions(t *testing.T) {
	h := testHandler(t)
	for _, tt := range []struct {
		action string
		want   string
	}{
		{"open Spotify", "Opening Spotify."},
		{"Open   VS Code", "Opening Visual Studio Code."},
		{"open terminal", "Opening Terminal."},
		{"open safari", "Opening Safari."},
		{"lock", "Locking your Mac."},
		{"sleep", "Putting your Mac to sleep."},
	} {
		got := speech(t, doHandle(t, h, request(t, RequestTypeIntent, IntentMacCommand,
			map[string]string{"Action": tt.action})))
		if got != tt.want {
			t.Errorf("action %q: speech = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestIntentUnknownAction(t *testing.T) {
	h := testHandler(t)
	got := speech(t, doHandle(t, h, request(t, RequestTypeIntent, IntentMacCommand,
		map[string]string{"Action": "delete everything"})))
	if !strings.Contains(got, "can't do that yet") {
		t.Errorf("unknown action speech = %q", got)
	}
}

func TestIntentHelpCancelStopFallback(t *testing.T) {
	h := testHandler(t)
	for _, tt := range []struct {
		intent string
		want   string
	}{
		{IntentHelp, "I can open apps"},
		{IntentCancel, "Goodbye."},
		{IntentStop, "Goodbye."},
		{IntentFallback, "I didn't catch that"},
	} {
		got := speech(t, doHandle(t, h, request(t, RequestTypeIntent, tt.intent, nil)))
		if !strings.Contains(got, tt.want) {
			t.Errorf("intent %s: speech = %q, want containing %q", tt.intent, got, tt.want)
		}
	}
}

func TestLaunchRequest(t *testing.T) {
	h := testHandler(t)
	got := speech(t, doHandle(t, h, request(t, RequestTypeLaunch, "", nil)))
	if !strings.Contains(got, "control your Mac") {
		t.Errorf("launch speech = %q", got)
	}
}

func TestServeHTTPUnauthenticated(t *testing.T) {
	reg := commands.NewRegistry()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(logger, NewVerifier("amzn1.ask.skill.test"), reg)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/alexa", strings.NewReader(`{}`)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestServeHTTPHappyPath(t *testing.T) {
	h := testHandler(t)
	req := request(t, RequestTypeIntent, IntentMacStatus, nil)
	raw, _ := json.Marshal(req)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/alexa", strings.NewReader(string(raw))))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if !strings.Contains(resp.Response.OutputSpeech.Text, "Your Mac is online") {
		t.Errorf("speech = %q", resp.Response.OutputSpeech.Text)
	}
}
