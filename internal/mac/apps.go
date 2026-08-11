package mac

import (
	"context"
	"errors"
	"strings"
)

// Bundle identifiers for the applications the controller can open. They are
// stable Apple bundle IDs, chosen instead of display names so the exact
// installed bundle never has to be guessed (PRD §7.2).
const (
	BundleSpotify  = "com.spotify.client"
	BundleVSCode   = "com.microsoft.VSCode"
	BundleTerminal = "com.apple.Terminal"
	BundleSafari   = "com.apple.Safari"
)

// OpenApp opens the application registered under the given bundle ID. The
// bundle path is detected through Spotlight rather than assumed.
func OpenApp(ctx context.Context, r Runner, bundleID string) error {
	path, err := findAppPath(ctx, r, bundleID)
	if err != nil {
		return err
	}
	return r.Run(ctx, "open", path)
}

func findAppPath(ctx context.Context, r Runner, bundleID string) (string, error) {
	query := "kMDItemCFBundleIdentifier == '" + bundleID + "'"
	out, err := r.Output(ctx, "mdfind", query)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if path := strings.TrimSpace(line); path != "" {
			return path, nil
		}
	}
	return "", errors.New("application not installed")
}
