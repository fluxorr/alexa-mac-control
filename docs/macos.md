# macOS Setup

Requirements: macOS 13+ (developed on macOS Tahoe 26, Apple Silicon), Go 1.24+.

## 1. Build the controller

```sh
make build
```

This produces `bin/server` (default port 2014).

## 2. Permissions

The controller runs ordinary macOS commands (`open`, `mdfind`, `pmset`,
`sysctl`, `vm_stat`, `df`, `top`, `shortcuts`). On modern macOS, **Terminal
or the parent process** must be granted:

- **Automation** (System Settings → Privacy & Security → Automation) if you
  use Shortcuts, so that `shortcuts run` may drive other apps.
- **Spotlight** permission is granted via the terminal's own prompts the
  first time `mdfind` runs.

No admin rights are required for any built-in command: nothing uses `sudo`.

## 3. macOS Shortcuts

Create shortcuts with these exact names (used by `coding_mode`; other
commands run native commands and do not need shortcuts, PRD §16):

| Shortcut name | Actions |
|---|---|
| `Mac - Coding Mode` | Open VS Code, open Terminal, open your project, open GitHub, play music — whatever you want; set `SHORTCUT_CODING_MODE=Mac - Coding Mode` |

To add a shortcut:
1. Open the **Shortcuts** app → **All Shortcuts** → **+**.
2. Add actions (e.g. **Open App** for VS Code, **Open Folder** for your
   project).
3. Run it once manually so macOS records the Automation consent.

## 4. Spotlight

File search uses Spotlight (`mdfind`). If search results seem stale:

```sh
sudo mdutil -E /
```

(That is the only command in this project that ever needs `sudo`, and it is
an optional maintenance step you run by hand — the server never runs it.)

## 5. Screen lock

`lock` uses the built-in `CGSession -suspend` helper. If the screen requires
a password after sleep (System Settings → Lock Screen → "Require password"),
this produces a true lock.

## 6. Sleep

`sleep` runs `pmset sleepnow`, which needs no privileges.

## 7. Environment

See `.env.example`. All configuration is via environment variables; nothing
is hardcoded. Export them in your shell before `make run`, or use a wrapper
script (`.env` is never committed).
