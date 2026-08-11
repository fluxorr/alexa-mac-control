package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fluxorr/alexa-mac-control/internal/mac"
	"github.com/fluxorr/alexa-mac-control/internal/security"
)

// Defaults carries the dependencies the built-in commands need. It is wired
// once at startup; the registry only ever invokes allowlisted actions.
type Defaults struct {
	Runner       mac.Runner
	SearchEngine mac.SearchEngine
	SearchRoots  []string
}

// maxNamedHits bounds how many file names the Alexa response names out loud
// (PRD §9: "The first three are ...").
const maxNamedHits = 3

// RegisterDefaults installs the built-in allowlisted commands (PRD §5-§9,
// §14). Every execute function is fixed at compile time: no command takes
// free-form input that reaches a shell.
func RegisterDefaults(r *Registry, d Defaults) {
	r.Register(&Command{
		Name:        "open_spotify",
		Description: "Open Spotify",
		Execute: func(ctx context.Context, _ CommandArgs) (Result, error) {
			return openApp(ctx, d.Runner, mac.BundleSpotify, "Spotify")
		},
	})
	r.Register(&Command{
		Name:        "open_vscode",
		Description: "Open Visual Studio Code",
		Execute: func(ctx context.Context, _ CommandArgs) (Result, error) {
			return openApp(ctx, d.Runner, mac.BundleVSCode, "Visual Studio Code")
		},
	})
	r.Register(&Command{
		Name:        "open_terminal",
		Description: "Open Terminal",
		Execute: func(ctx context.Context, _ CommandArgs) (Result, error) {
			return openApp(ctx, d.Runner, mac.BundleTerminal, "Terminal")
		},
	})
	r.Register(&Command{
		Name:        "open_safari",
		Description: "Open Safari",
		Execute: func(ctx context.Context, _ CommandArgs) (Result, error) {
			return openApp(ctx, d.Runner, mac.BundleSafari, "Safari")
		},
	})
	r.Register(&Command{
		Name:        "search_web",
		Description: "Search the web",
		Execute: func(ctx context.Context, args CommandArgs) (Result, error) {
			query := args["query"]
			if err := security.ValidateQuery(query); err != nil {
				return Result{}, err
			}
			if err := mac.OpenWebSearch(ctx, d.Runner, d.SearchEngine, query); err != nil {
				return Result{}, err
			}
			return Result{Message: fmt.Sprintf("Searching for %s.", query)}, nil
		},
	})
	r.Register(&Command{
		Name:        "search_files",
		Description: "Search files on this Mac",
		Execute: func(ctx context.Context, args CommandArgs) (Result, error) {
			query := args["query"]
			if err := security.ValidateQuery(query); err != nil {
				return Result{}, err
			}
			if len(d.SearchRoots) == 0 {
				return Result{Message: "File search is not configured on this Mac."}, nil
			}
			hits, err := mac.SearchFiles(ctx, d.Runner, query, d.SearchRoots)
			if err != nil {
				return Result{}, err
			}
			switch {
			case len(hits) == 0:
				return Result{Message: "I couldn't find anything matching that."}, nil
			case len(hits) == 1:
				return Result{Message: fmt.Sprintf("I found one file: %s.", filepath.Base(hits[0])), Data: hits}, nil
			default:
				names := make([]string, 0, maxNamedHits)
				for _, h := range hits[:min(maxNamedHits, len(hits))] {
					names = append(names, filepath.Base(h))
				}
				return Result{
					Message: fmt.Sprintf("I found %d matching files. The first %d are %s.",
						len(hits), len(names), joinNames(names)),
					Data: hits,
				}, nil
			}
		},
	})
	r.Register(&Command{
		Name:        "system_status",
		Description: "Report system status",
		Execute: func(ctx context.Context, _ CommandArgs) (Result, error) {
			st, err := mac.SystemStatus(ctx, d.Runner)
			if err != nil {
				return Result{}, err
			}
			return Result{
				Message: fmt.Sprintf("Your Mac is online. CPU usage is %.0f percent, memory usage is %.0f percent, and battery is at %d percent.",
					st.CPU, st.Memory, st.Battery),
				Data: st,
			}, nil
		},
	})
}

func openApp(ctx context.Context, r mac.Runner, bundleID, displayName string) (Result, error) {
	if err := mac.OpenApp(ctx, r, bundleID); err != nil {
		return Result{}, err
	}
	return Result{Message: fmt.Sprintf("Opening %s.", displayName)}, nil
}

// joinNames renders a list for speech: "a", "a and b", "a, b, and c".
func joinNames(names []string) string {
	switch len(names) {
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
}
