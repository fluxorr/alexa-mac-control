package mac

import "context"

// RunShortcut invokes a macOS Shortcut by name (PRD §16). The name is a
// fixed registry value defined at the command layer, never free text.
func RunShortcut(ctx context.Context, r Runner, name string) error {
	return r.Run(ctx, "shortcuts", "run", name)
}
