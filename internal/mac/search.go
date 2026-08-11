package mac

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// SearchEngine selects the web search backend (PRD §8).
type SearchEngine string

const (
	EngineGoogle     SearchEngine = "google"
	EngineDuckDuckGo SearchEngine = "duckduckgo"
)

// OpenWebSearch opens the configured search engine with the query in the
// user's default browser. The URL is built here from a fixed template; a
// query is never treated as a URL (PRD §8).
func OpenWebSearch(ctx context.Context, r Runner, engine SearchEngine, query string) error {
	u, ok := searchURL(engine, query)
	if !ok {
		return fmt.Errorf("unsupported search engine %q", engine)
	}
	return r.Run(ctx, "open", u)
}

func searchURL(engine SearchEngine, query string) (string, bool) {
	q := url.QueryEscape(query)
	switch engine {
	case EngineGoogle:
		return "https://www.google.com/search?q=" + q, true
	case EngineDuckDuckGo:
		return "https://duckduckgo.com/?q=" + q, true
	}
	return "", false
}

// maxFileResults bounds how many matches are ever returned to callers;
// Alexa speech must stay short (PRD §9).
const maxFileResults = 10

// SearchFiles runs a Spotlight search restricted to the given roots and
// returns at most maxFileResults unique paths. An empty root list searches
// nothing: the whole filesystem is never searched by default (PRD §9).
func SearchFiles(ctx context.Context, r Runner, query string, roots []string) ([]string, error) {
	if len(roots) == 0 {
		return nil, nil
	}

	hits := make([]string, 0, maxFileResults)
	seen := make(map[string]bool, maxFileResults)
	for _, root := range roots {
		out, err := r.Output(ctx, "mdfind", "-onlyin", root, query)
		if err != nil {
			return nil, fmt.Errorf("search %s: %w", root, err)
		}
		for _, line := range strings.Split(out, "\n") {
			path := strings.TrimSpace(line)
			// mdfind may interleave stderr diagnostics (e.g. query-parser
			// notices) into combined output; only absolute paths are hits.
			if path == "" || !strings.HasPrefix(path, "/") || seen[path] {
				continue
			}
			seen[path] = true
			hits = append(hits, path)
			if len(hits) >= maxFileResults {
				return hits, nil
			}
		}
	}
	return hits, nil
}
