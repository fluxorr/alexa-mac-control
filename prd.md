# PRD — Mac Voice Control via Alexa

## 1. Product Overview

Build a secure, local-first system that allows an Amazon Echo Dot 3rd generation to control a MacBook through an Alexa Custom Skill.

The MacBook is an Apple Silicon Mac running macOS Tahoe 26.x.

The system should allow natural voice commands such as:

* "Alexa, ask Mac to open Spotify."
* "Alexa, ask Mac to open VS Code."
* "Alexa, ask Mac to search for Go closures."
* "Alexa, ask Mac to search my files for authentication middleware."
* "Alexa, ask Mac to show system status."
* "Alexa, ask Mac to lock."
* "Alexa, ask Mac to sleep."
* "Alexa, ask Mac to restart."
* "Alexa, ask Mac to open my backend project."
* "Alexa, ask Mac to start coding mode."

The Echo Dot itself must remain completely untouched.

Do NOT open, modify, root, flash, solder, or otherwise physically modify the Echo Dot.

---

# 2. Core Goal

Create a bridge between:

```text
Echo Dot
    ↓
Alexa
    ↓
Alexa Custom Skill
    ↓
HTTPS
    ↓
Go Mac Controller
    ↓
macOS Shortcuts / AppleScript / safe system commands
    ↓
Mac
```

The first version should NOT use an LLM.

The architecture must, however, make it possible to add an LLM later.

---

# 3. Technology Stack

Use:

* Go
* Go standard library wherever practical
* Alexa Custom Skill
* macOS Shortcuts
* AppleScript only where necessary
* macOS shell commands only where necessary
* Cloudflare Tunnel for development/public HTTPS access
* JSON for internal APIs
* structured logging
* unit tests

Do NOT introduce a large framework unless there is a clear reason.

Do NOT use Home Assistant for the initial implementation.

Do NOT add a database unless it becomes necessary.

Do NOT add an LLM in the first version.

---

# 4. Repository Structure

Create a clean Go project with approximately this structure:

```text
mac-alexa/
├── AGENTS.md
├── README.md
├── go.mod
├── go.sum
│
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── alexa/
│   │   ├── handler.go
│   │   ├── models.go
│   │   ├── intents.go
│   │   └── response.go
│   │
│   ├── commands/
│   │   ├── command.go
│   │   ├── registry.go
│   │   └── executor.go
│   │
│   ├── mac/
│   │   ├── shortcuts.go
│   │   ├── apps.go
│   │   ├── system.go
│   │   └── search.go
│   │
│   ├── security/
│   │   ├── auth.go
│   │   └── validation.go
│   │
│   └── config/
│       └── config.go
│
├── scripts/
│   └── setup.sh
│
└── docs/
    ├── alexa.md
    ├── macos.md
    ├── cloudflare.md
    └── security.md
```

Adjust the structure if necessary, but keep the separation of concerns.

---

# 5. Security Principle

This is the most important requirement.

NEVER create an endpoint that accepts arbitrary shell commands.

Do NOT implement:

```text
POST /execute
{
    "command": "rm -rf ..."
}
```

Do NOT execute Alexa's raw text using:

```go
exec.Command("sh", "-c", userInput)
```

Do NOT allow an LLM to directly execute shell commands in the future.

Every operation must go through an explicit allowlisted command/action.

Example:

```text
OPEN_SPOTIFY
OPEN_VSCODE
OPEN_TERMINAL
OPEN_SAFARI
OPEN_BACKEND
SEARCH_WEB
SEARCH_FILES
LOCK
SLEEP
RESTART
SYSTEM_STATUS
CODING_MODE
```

The executor must reject unknown commands.

---

# 6. Command Architecture

Create a command abstraction.

Conceptually:

```go
type Command struct {
    Name        string
    Description string
    Execute     func(ctx context.Context, args CommandArgs) (CommandResult, error)
}
```

Use an explicit registry.

Example:

```text
OPEN_SPOTIFY
OPEN_VSCODE
OPEN_TERMINAL
OPEN_SAFARI
OPEN_BACKEND
LOCK
SLEEP
RESTART
SYSTEM_STATUS
SEARCH_WEB
SEARCH_FILES
```

The Alexa layer should never know how a command is executed.

The Mac layer should never know anything about Alexa.

The command layer should sit between them.

Architecture:

```text
Alexa
  ↓
Intent
  ↓
Command
  ↓
Executor
  ↓
Mac implementation
```

---

# 7. Initial Commands

Implement the following.

## 7.1 Open Spotify

Voice:

> "Alexa, ask Mac to open Spotify."

Implementation may use:

```bash
open -a "Spotify"
```

Prefer a Shortcut if appropriate.

---

## 7.2 Open VS Code

Voice:

> "Alexa, ask Mac to open VS Code."

Use the installed application.

Do not assume the exact application bundle name without detecting it.

---

## 7.3 Open Terminal

Open macOS Terminal.

---

## 7.4 Open Safari

Open Safari.

---

## 7.5 Open Development Project

Create a configurable development directory.

Example:

```text
~/Developer
```

Allow configuration through environment variables or config.

Do not hardcode the user's real project paths.

The command should be able to open the configured project in Finder and/or VS Code.

---

# 8. Web Search

Support:

> "Alexa, ask Mac to search for Go closures."

The Alexa request should extract:

```json
{
  "query": "Go closures"
}
```

The Go server should construct the search URL safely.

Do NOT pass arbitrary URLs directly from Alexa.

Default search engine should be configurable.

For example:

```text
SEARCH_ENGINE=google
```

or:

```text
SEARCH_ENGINE=duckduckgo
```

Use URL encoding correctly.

Open the result in the user's default browser.

---

# 9. File Search

Support:

> "Alexa, ask Mac to search my files for authentication middleware."

Initial implementation should search only configured directories.

Example:

```text
SEARCH_ROOTS=~/Developer,~/Documents
```

Do NOT allow Alexa to search the entire filesystem by default.

Use a safe local search mechanism.

Potential implementations include:

* `mdfind`
* Spotlight
* controlled filesystem traversal

Prefer `mdfind` where appropriate on macOS.

Return a concise result suitable for Alexa speech.

For example:

> "I found 7 matching files. The first three are handler.go, auth.go, and middleware.go."

Do not return huge file lists.

---

# 10. Lock Mac

Support:

> "Alexa, ask Mac to lock."

Use an appropriate macOS-native mechanism.

This should be implemented as a dedicated command.

Do not expose arbitrary shell execution.

---

# 11. Sleep Mac

Support:

> "Alexa, ask Mac to sleep."

This should execute a dedicated, explicit sleep operation.

---

# 12. Restart Mac

Support:

> "Alexa, ask Mac to restart."

This is a destructive/system-level action.

Require an additional confirmation mechanism if possible.

Do not execute it based on ambiguous intent.

---

# 13. Shutdown

Shutdown should NOT be enabled in the first MVP unless the confirmation flow is implemented.

Treat shutdown as a high-risk command.

Design the command system so it can later support:

```text
REQUEST_SHUTDOWN
CONFIRM_SHUTDOWN
```

rather than immediately executing shutdown.

---

# 14. System Status

Support:

> "Alexa, ask Mac for system status."

Return useful but concise information:

* Mac online/offline
* uptime
* CPU usage
* memory usage
* battery percentage
* charging state
* disk usage

Example Alexa response:

> "Your Mac is online. CPU usage is 18 percent, memory usage is 62 percent, and battery is at 74 percent."

Do not return excessive information.

---

# 15. Coding Mode

Create a configurable automation called:

```text
CODING_MODE
```

Initially this can execute predefined Shortcuts.

Example:

```text
Open VS Code
Open Terminal
Open project
Open GitHub
Start preferred music
```

Do NOT hardcode the user's personal applications or directories.

Make these configurable.

The Shortcut should ultimately be responsible for the workflow where practical.

---

# 16. macOS Shortcuts Integration

Use macOS Shortcuts as a safe automation layer.

Create documentation and optionally setup scripts for:

```text
Mac - Open Spotify
Mac - Open VS Code
Mac - Open Terminal
Mac - Open Safari
Mac - Open Backend
Mac - Coding Mode
Mac - Lock
Mac - Sleep
Mac - System Status
```

The Go server should be able to invoke shortcuts through:

```bash
shortcuts run "Shortcut Name"
```

Do not assume all shortcuts already exist.

Provide clear setup documentation.

Where possible, make the application work even if Shortcuts are not used for simple native actions.

---

# 17. Alexa Custom Skill

Design the Alexa skill around a small number of intents.

Do not create dozens of intents.

Use intents similar to:

```text
MacCommandIntent
MacSearchIntent
MacFileSearchIntent
MacStatusIntent
AMAZON.HelpIntent
AMAZON.CancelIntent
AMAZON.StopIntent
AMAZON.FallbackIntent
```

Example utterances:

```text
ask Mac to open Spotify
ask Mac to open VS Code
ask Mac to lock
ask Mac to sleep
ask Mac for system status
ask Mac to search for {query}
ask Mac to search my files for {query}
```

Keep the invocation name simple.

Recommended:

```text
Mac
```

So:

> "Alexa, ask Mac to open Spotify."

---

# 18. Alexa Request Verification

Implement proper Alexa request verification.

The server must verify that incoming requests are genuinely from Alexa.

Do not trust arbitrary POST requests simply because they hit the endpoint.

Validate:

* request signature
* timestamp
* request structure
* application/skill ID
* request type

Follow Amazon's current Custom Skill request verification requirements.

Do not invent a custom verification mechanism when Alexa already provides one.

---

# 19. Authentication

The Mac control endpoint must not become an unrestricted public API.

Use a layered security model:

```text
Alexa
 ↓
Alexa request verification
 ↓
skill ID verification
 ↓
command allowlist
 ↓
argument validation
 ↓
Mac executor
```

For local development, bind the Mac controller to:

```text
127.0.0.1
```

not:

```text
0.0.0.0
```

unless explicitly required.

---

# 20. Cloudflare Tunnel

Document how to expose the local Go server through Cloudflare Tunnel.

Desired architecture:

```text
Alexa
  ↓
HTTPS
  ↓
Cloudflare
  ↓
Cloudflare Tunnel
  ↓
127.0.0.1:8787
```

Do not expose router ports.

Do not implement port forwarding.

Do not require a public IP.

Document the tunnel setup separately from the application.

The application itself should remain unaware of Cloudflare.

---

# 21. Configuration

Use environment variables for sensitive/configurable values.

Example:

```text
PORT=8787

ALEXA_SKILL_ID=
ALEXA_APP_ID=

SEARCH_ENGINE=google

DEVELOPER_ROOT=~/Developer
BACKEND_PROJECT=~/Developer/backend

SHORTCUT_OPEN_SPOTIFY=Mac - Open Spotify
SHORTCUT_OPEN_VSCODE=Mac - Open VS Code
SHORTCUT_CODING_MODE=Mac - Coding Mode
```

Do not commit secrets.

Provide:

```text
.env.example
```

but never commit:

```text
.env
```

---

# 22. Logging

Implement structured logs.

Log:

* request received
* Alexa request type
* intent
* resolved command
* command execution
* execution duration
* success/failure

Do NOT log:

* Alexa access tokens
* secrets
* sensitive file contents
* arbitrary command strings
* full private user data

---

# 23. Health Endpoint

Implement:

```text
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

Also optionally expose:

```text
GET /version
```

---

# 24. Local Development API

Before integrating Alexa, the entire system must be testable locally.

Create:

```text
POST /api/command
```

Example:

```json
{
  "command": "open_spotify"
}
```

Expected behavior:

```text
Go server
 ↓
command registry
 ↓
Spotify opens
```

This endpoint is for LOCAL DEVELOPMENT ONLY.

It must not be exposed through the public Alexa endpoint.

---

# 25. Testing

Write unit tests for:

* Alexa request parsing
* Alexa request verification
* intent mapping
* command lookup
* unknown command rejection
* search query validation
* URL encoding
* file-search restrictions
* authorization
* command execution
* Alexa response generation

Use mocks/interfaces around OS-level execution.

Tests must not accidentally:

* shut down the Mac
* restart the Mac
* sleep the Mac
* modify user files
* open applications

Create safe fake executors for tests.

---

# 26. Error Handling

Alexa responses must be human-friendly.

Bad:

> "error: exec.Command exit status 1"

Good:

> "I couldn't open Spotify."

Bad:

> "mdfind returned code 1"

Good:

> "I couldn't find anything matching that."

Do not expose internal implementation details through Alexa.

---

# 27. Future LLM Architecture

Do not implement this now.

However, design the command layer so a future LLM can produce structured actions.

Future:

```text
Alexa
 ↓
Go API
 ↓
LLM
 ↓
Structured Action
 ↓
Policy / Allowlist
 ↓
Command Registry
 ↓
Mac
```

Example:

User:

> "Alexa, get my development environment ready."

LLM output:

```json
{
  "actions": [
    {
      "command": "open_vscode"
    },
    {
      "command": "open_backend"
    },
    {
      "command": "coding_mode"
    }
  ]
}
```

The policy layer must validate every action.

The LLM must NEVER receive direct shell execution privileges.

---

# 28. Future Natural Language Search

Eventually support:

> "Alexa, find that Go project where I was working on closures."

The future architecture may use:

```text
Alexa
 ↓
Go
 ↓
LLM
 ↓
structured search request
 ↓
local search
 ↓
results
 ↓
Alexa
```

This should be treated as a separate feature after the MVP.

---

# 29. OpenCode Instructions

You are being assisted by OpenCode.

Before writing significant code:

1. Inspect the repository.
2. Read existing files.
3. Create/update `AGENTS.md`.
4. Establish the architecture.
5. Create a short implementation plan.
6. Implement incrementally.
7. Run tests after each major component.
8. Never silently make security-sensitive architectural decisions.
9. Do not install unnecessary dependencies.
10. Prefer the Go standard library.

OpenCode should ask for confirmation before:

* exposing network services publicly
* modifying firewall/network settings
* installing system software
* changing macOS security settings
* enabling SSH/Remote Login
* creating privileged launch agents
* executing destructive commands

---

# 30. OpenCode Development Rules

Never execute these automatically:

```text
sudo
rm -rf
shutdown
reboot
diskutil erase
system modification
firewall modification
```

Never create arbitrary shell execution.

Never store credentials in source code.

Never expose a shell endpoint.

Never bypass macOS security mechanisms.

Never weaken Alexa request verification.

Never use an LLM to directly construct shell commands.

---

# 31. MVP Definition

The MVP is complete when the following works:

### Local

```text
POST /api/command
{
  "command": "open_spotify"
}
```

→ Spotify opens.

And:

```text
POST /api/command
{
  "command": "search_web",
  "query": "Go closures"
}
```

→ browser opens the search.

And:

```text
POST /api/command
{
  "command": "system_status"
}
```

→ returns structured system information.

### Alexa

The following spoken commands must work:

> "Alexa, ask Mac to open Spotify."

> "Alexa, ask Mac to open VS Code."

> "Alexa, ask Mac to search for Go closures."

> "Alexa, ask Mac for system status."

> "Alexa, ask Mac to lock."

> "Alexa, ask Mac to sleep."

---

# 32. Documentation Requirements

Create a high-quality README explaining:

1. What the project does
2. Architecture
3. Requirements
4. macOS setup
5. Shortcut setup
6. Go setup
7. Alexa Developer Console setup
8. Alexa skill configuration
9. HTTPS requirements
10. Cloudflare Tunnel setup
11. Environment variables
12. Running locally
13. Testing
14. Security model
15. Adding a new command
16. Troubleshooting

Also create:

```text
docs/alexa.md
docs/macos.md
docs/cloudflare.md
docs/security.md
```

---

# 33. Adding New Commands

Make adding a command straightforward.

A developer should be able to:

```text
1. Add command definition.
2. Add executor.
3. Add Alexa utterances if required.
4. Add tests.
5. Update documentation.
```

Do not require modifying unrelated components.

---

# 34. Design Philosophy

Keep the system:

* local-first
* minimal
* secure
* modular
* observable
* easy to extend
* easy to debug
* easy to remove

The Echo should be treated as the voice interface.

The Mac should be treated as the execution environment.

The Go application should be the security and orchestration layer.

---

# 35. Final Architecture

The intended final system is:

```text
                    ┌───────────────┐
                    │  Echo Dot 3   │
                    └───────┬───────┘
                            │
                          Alexa
                            │
                            ▼
                  ┌──────────────────┐
                  │  Alexa Custom    │
                  │      Skill       │
                  └────────┬─────────┘
                           │
                         HTTPS
                           │
                           ▼
                  ┌──────────────────┐
                  │ Cloudflare       │
                  │ Tunnel            │
                  └────────┬─────────┘
                           │
                           ▼
                  ┌──────────────────┐
                  │   Go Controller  │
                  │                  │
                  │ Request Verify   │
                  │ Intent Parser    │
                  │ Command Registry │
                  │ Policy Layer     │
                  └────────┬─────────┘
                           │
              ┌────────────┼─────────────┐
              │            │             │
              ▼            ▼             ▼
          Shortcuts    macOS APIs      Safe CLI
              │            │             │
              └────────────┼─────────────┘
                           ▼
                    ┌─────────────┐
                    │ macOS Tahoe │
                    │ MacBook Air │
                    └─────────────┘
```

---

# 36. Implementation Order

Implement exactly in this order:

### Step 1

Repository + `AGENTS.md`

### Step 2

Go HTTP server

### Step 3

Command abstraction + registry

### Step 4

Mac executor

### Step 5

Safe commands:

```text
open_spotify
open_vscode
open_terminal
open_safari
search_web
system_status
```

### Step 6

Lock + sleep

### Step 7

Unit tests

### Step 8

Alexa request/response models

### Step 9

Alexa intent handling

### Step 10

Alexa request verification

### Step 11

Alexa Developer Console integration

### Step 12

Cloudflare Tunnel

### Step 13

End-to-end Alexa testing

### Step 14

Improve natural language handling

### Step 15

Add additional commands

### Step 16

Only after everything is stable, design the LLM layer.

---

# 37. Success Criteria

The project should feel like this:

```text
Me:
"Alexa, ask Mac to open VS Code."

Alexa:
"Okay."

Mac:
VS Code opens.
```

Then:

```text
Me:
"Alexa, ask Mac to search for Go closures."

Mac:
Browser opens with the search.

Alexa:
"Searching for Go closures."
```

And:

```text
Me:
"Alexa, ask Mac for system status."

Alexa:
"Your Mac is online. CPU usage is 14 percent,
memory usage is 58 percent, and battery is 81 percent."
```

The entire system should accomplish this **without modifying the Echo Dot hardware**.

---

# 38. Important Constraint

Do not over-engineer the first version.

The goal is not to build JARVIS immediately.

The goal is to build a **secure, reliable Alexa → Mac command pipeline**.

Once that works, the system can progressively evolve into:

```text
Alexa
 ↓
Mac Controller
 ↓
LLM
 ↓
Tools
 ↓
Mac
```

without changing the fundamental architecture.
