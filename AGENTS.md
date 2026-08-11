# AGENTS.md — Alexa Mac Control

## Project
Secure, local-first bridge: Echo Dot → Alexa Custom Skill → HTTPS → Go controller → macOS (Shortcuts / AppleScript / safe CLI). See `prd.md` (source of truth) and `TODO.md` (progress).

## Hard rules (from PRD §5, §30)
- NEVER arbitrary shell execution. No `POST /execute` with a command string. No `exec.Command("sh", "-c", userInput)`.
- Every action goes through an explicit allowlisted command in the registry. Unknown commands are rejected.
- Alexa layer never knows how commands run; Mac layer never knows about Alexa.
- No LLM, no database, no Home Assistant in v1. Go stdlib first; no frameworks without a clear reason.
- Never weaken Alexa request verification (signature, timestamp, skill ID).
- Never store credentials in source. Bind to 127.0.0.1, not 0.0.0.0.
- Never run: sudo, rm -rf, shutdown, reboot, diskutil erase, firewall/system modification.
- OpenCode must ask before: exposing network services publicly, modifying firewall/network, installing system software, changing macOS security settings, enabling SSH, privileged launch agents, destructive commands.

## Workflow
1. Inspect repo, read existing files before writing code.
2. Update `TODO.md` after every change/discussion (status legend at top).
3. Implement incrementally per PRD §36 order; run tests after each major component.
4. Commit every small, meaningful change. Do NOT commit agent-specific files (`.opencode/`, `.agents/`, scratch, `agent-*.md` — gitignored).
5. Keep AGENTS.md and TODO.md current in the same commit as the work they describe.

## Architecture (PRD §35)
```
Echo Dot → Alexa → Custom Skill → HTTPS → Cloudflare Tunnel → Go Controller
(Request Verify → Intent Parser → Command Registry → Policy Layer)
→ Shortcuts / macOS APIs / Safe CLI → macOS
```
Future (not now): LLM produces structured actions validated by the policy layer, never direct shell access.

## Implementation order (PRD §36)
repo → HTTP server → command abstraction+registry → Mac executor → safe commands → lock+sleep → unit tests → Alexa models → intent handling → request verification → Dev Console → Cloudflare Tunnel → E2E → improve NL → more commands → LLM last.

## Testing
Unit tests with mocked OS exec layer. Tests must never: shut down/restart/sleep the Mac, modify files, or open apps. Use fake executors.

## Conventions
- Human-friendly Alexa responses, never raw error text (PRD §26).
- Structured logging; never log tokens, secrets, raw command strings, file contents.
- `.env.example` documented, `.env` never committed.
- Config via env vars: `PORT`, `ALEXA_SKILL_ID`, `SEARCH_ENGINE`, `DEVELOPER_ROOT`, `BACKEND_PROJECT`, `SHORTCUT_*`.
