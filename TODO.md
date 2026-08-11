# TODO — Alexa Mac Control

Status legend: `[ ]` pending · `[~]` in progress · `[x]` done

## Step 1 — Repository + AGENTS.md
- [x] Git repo + GitHub (fluxorr/alexa-mac-control, public)
- [x] .gitignore (agent files, .env)
- [x] AGENTS.md
- [x] TODO.md
- [ ] README.md (final polish per PRD §32, written incrementally)

## Step 2 — Go HTTP server
- [ ] go.mod, cmd/server/main.go
- [ ] GET /health, GET /version
- [ ] structured logging
- [ ] config via env (PORT, ALEXA_SKILL_ID, SEARCH_ENGINE, DEVELOPER_ROOT, ...)

## Step 3 — Command abstraction + registry
- [ ] internal/commands/command.go, registry.go
- [ ] internal/commands/executor.go
- [ ] internal/security/validation.go, auth.go
- [ ] POST /api/command (local dev only)

## Step 4 — Mac executor
- [ ] internal/mac/apps.go (open app, detect bundle name)
- [ ] internal/mac/system.go (lock, sleep, status)
- [ ] internal/mac/search.go (web + file via mdfind)
- [ ] internal/mac/shortcuts.go (shortcuts run)
- [ ] safe fake executors for tests

## Step 5 — Safe commands
- [ ] open_spotify
- [ ] open_vscode
- [ ] open_terminal
- [ ] open_safari
- [ ] search_web (SEARCH_ENGINE config, URL encoding)
- [ ] system_status (uptime, cpu, mem, battery, disk)

## Step 6 — Lock + sleep
- [ ] lock
- [ ] sleep
- [ ] (restart/shutdown deferred — needs confirmation flow, not in MVP)

## Step 7 — Unit tests
- [ ] command lookup / unknown rejection
- [ ] search query validation + URL encoding
- [ ] file-search restrictions
- [ ] exec layer mocked, no destructive actions in tests

## Step 8 — Alexa request/response models
- [ ] internal/alexa/models.go, response.go

## Step 9 — Alexa intent handling
- [ ] internal/alexa/intents.go, handler.go
- [ ] MacCommandIntent, MacSearchIntent, MacFileSearchIntent, MacStatusIntent
- [ ] Help/Cancel/Stop/Fallback

## Step 10 — Alexa request verification
- [ ] signature, timestamp, skill ID, request type

## Step 11 — Alexa Developer Console integration
- [ ] skill "Mac", invocation name, utterances
- [ ] docs/alexa.md

## Step 12 — Cloudflare Tunnel
- [ ] docs/cloudflare.md
- [ ] tunnel setup (no port forwarding)

## Step 13 — End-to-end Alexa testing
- [ ] full loop: voice → skill → tunnel → Go → Mac

## Step 14+ — Later
- [ ] improve natural language handling
- [ ] additional commands (open_backend, coding_mode)
- [ ] restart with confirmation flow
- [ ] LLM layer (only after stable, §27/§36.16)

## Docs (PRD §32)
- [ ] docs/macos.md
- [ ] docs/security.md
- [ ] README full pass
