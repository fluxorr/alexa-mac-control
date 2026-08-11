package mac

import (
	"context"
	"fmt"
	"testing"
)

func TestOpenWebSearchGoogle(t *testing.T) {
	fake := newFakeRunner()
	if err := OpenWebSearch(context.Background(), fake, EngineGoogle, "go closures"); err != nil {
		t.Fatalf("OpenWebSearch error = %v", err)
	}
	if !fake.ran("open", "https://www.google.com/search?q=go+closures") {
		t.Errorf("unexpected open call; calls = %v", fake.calls)
	}
}

func TestOpenWebSearchDuckDuckGo(t *testing.T) {
	fake := newFakeRunner()
	if err := OpenWebSearch(context.Background(), fake, EngineDuckDuckGo, "a&b+c"); err != nil {
		t.Fatalf("OpenWebSearch error = %v", err)
	}
	if !fake.ran("open", "https://duckduckgo.com/?q=a%26b%2Bc") {
		t.Errorf("query was not URL-escaped correctly; calls = %v", fake.calls)
	}
}

func TestOpenWebSearchUnknownEngine(t *testing.T) {
	fake := newFakeRunner()
	if err := OpenWebSearch(context.Background(), fake, SearchEngine("bing"), "x"); err == nil {
		t.Error("OpenWebSearch: want error for unsupported engine")
	}
	if fake.ran("open") {
		t.Error("open must not be called for an unsupported engine")
	}
}

func TestSearchFiles(t *testing.T) {
	fake := newFakeRunner().
		withOutput("mdfind -onlyin /Users/me/Developer go closures",
			"/Users/me/Developer/a.go\n/Users/me/Developer/b.go\n\n").
		withOutput("mdfind -onlyin /Users/me/Documents go closures",
			"/Users/me/Documents/c.md\n/Users/me/Developer/a.go\n")

	hits, err := SearchFiles(context.Background(), fake, "go closures",
		[]string{"/Users/me/Developer", "/Users/me/Documents"})
	if err != nil {
		t.Fatalf("SearchFiles error = %v", err)
	}
	want := []string{
		"/Users/me/Developer/a.go",
		"/Users/me/Developer/b.go",
		"/Users/me/Documents/c.md",
	}
	if len(hits) != len(want) {
		t.Fatalf("hits = %v, want %v", hits, want)
	}
	for i := range want {
		if hits[i] != want[i] {
			t.Errorf("hits[%d] = %q, want %q", i, hits[i], want[i])
		}
	}
}

func TestSearchFilesNoRoots(t *testing.T) {
	fake := newFakeRunner()
	hits, err := SearchFiles(context.Background(), fake, "anything", nil)
	if err != nil {
		t.Fatalf("SearchFiles error = %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("hits = %v, want none without roots", hits)
	}
	if len(fake.calls) != 0 {
		t.Errorf("no commands must run without roots; calls = %v", fake.calls)
	}
}

func TestSearchFilesLimited(t *testing.T) {
	var many string
	for i := 0; i < maxFileResults+5; i++ {
		many += fmt.Sprintf("/Users/me/Developer/file%d.go\n", i)
	}
	fake := newFakeRunner().withOutput("mdfind -onlyin /Users/me/Developer query", many)

	hits, err := SearchFiles(context.Background(), fake, "query", []string{"/Users/me/Developer"})
	if err != nil {
		t.Fatalf("SearchFiles error = %v", err)
	}
	if len(hits) != maxFileResults {
		t.Errorf("len(hits) = %d, want %d", len(hits), maxFileResults)
	}
}
