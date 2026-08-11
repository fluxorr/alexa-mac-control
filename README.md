# Alexa Mac Control

Speak to your Mac through an Echo Dot. A secure, local-first bridge:

```
Echo Dot → Alexa → Custom Skill → HTTPS → Cloudflare Tunnel → Go Controller
(Request Verify → Intent Parser → Command Registry → Policy Layer)
→ Shortcuts / macOS APIs / Safe CLI → macOS
```

The Echo Dot is untouched; nothing listens on the public internet except a
Cloudflare Tunnel; the controller accepts **only** allowlisted commands and
verifies that every request genuinely came from Alexa.

## What you can say

- "Alexa, ask Mac to open Spotify." / "…open VS Code." / "…open Terminal." / "…open Safari."
- "Alexa, ask Mac to open the backend project."
- "Alexa, ask Mac to start coding mode."
- "Alexa, ask Mac to search for Go closures."
- "Alexa, ask Mac to search my files for authentication middleware."
- "Alexa, ask Mac for system status."
- "Alexa, ask Mac to lock." / "…to sleep."

## Architecture

| Layer | Responsibility |
|---|---|
| Echo Dot / Alexa | voice interface only |
| Custom Skill | fixed intents: `MacCommandIntent`, `MacSearchIntent`, `MacFileSearchIntent`, `MacStatusIntent` |
| Cloudflare Tunnel | public HTTPS → `127.0.0.1:2014`; no ports opened |
| Go controller | request verification, intent parsing, command registry, policy |
| macOS | Shortcuts, `open`, `pmset`, `mdfind`, `sysctl`, `vm_stat` — explicit argv, never a shell |

## Requirements

- macOS 13+ (built on macOS Tahoe 26, Apple Silicon)
- Go 1.24+
- An Amazon developer account and an Echo device (for the skill)
- A Cloudflare account (free) for the tunnel

## Setup

1. **macOS** — see [docs/macos.md](docs/macos.md): permissions, the
   `Mac - Coding Mode` shortcut, Spotlight.
2. **Go server**:

   ```sh
   make build
   cp .env.example .env    # edit values
   set -a; source .env; set +a   # or export by hand
   make run                # listens on 127.0.0.1:2014
   ```

3. **Alexa skill** — see [docs/alexa.md](docs/alexa.md): create the skill,
   add the four intents, set the HTTPS endpoint and `ALEXA_SKILL_ID`.
4. **Cloudflare Tunnel** — see [docs/cloudflare.md](docs/cloudflare.md):
   point a public URL at the loopback server.

## Environment variables

See `.env.example`. Key ones:

| Variable | Meaning | Default |
|---|---|---|
| `HOST` | bind address (loopback only; `0.0.0.0` is rejected) | `127.0.0.1` |
| `PORT` | listen port | `2014` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `SEARCH_ENGINE` | `google` / `duckduckgo` | `google` |
| `SEARCH_ROOTS` | comma list of folders Spotlight may search (`~` expands) | empty (disabled) |
| `ALEXA_SKILL_ID` | skill application ID; unset disables `/alexa` | empty |
| `DEVELOPER_ROOT` | folder opened by "open the backend project" | empty (disabled) |
| `SHORTCUT_CODING_MODE` | Shortcut run by "start coding mode" | empty (disabled) |

## Running locally

```sh
make run
```

Check the API without Alexa:

```sh
curl http://127.0.0.1:2014/health            # {"status":"ok"}
curl -X POST http://127.0.0.1:2014/api/command \
     -d '{"command":"system_status"}'         # JSON status + speech text
curl -X POST http://127.0.0.1:2014/api/command \
     -d '{"command":"search_web","query":"go closures"}'  # opens browser
```

`POST /api/command` is for **local development only** and never reaches the
public tunnel.

## Testing

```sh
make test    # go test ./...
make vet
```

All OS interaction goes through a `Runner` interface; tests inject fakes, so
the suite can never lock, sleep, open apps, or modify your files. The Alexa
verification tests sign requests with a locally generated certificate chain.

## Security model

- Every action is an allowlisted command; unknown commands are rejected.
- No shell anywhere: explicit argv only, never `sh -c`.
- Alexa requests are verified: certificate URL, chain trust, signature,
  timestamp (±150 s), skill ID.
- Loopback-only bind enforced at startup; `/api/command` never exposed.
- No secrets in source; `.env` gitignored.
- Restart/shutdown are intentionally not implemented (they need a
  confirmation flow, PRD §12-§13).

Full detail: [docs/security.md](docs/security.md).

## Adding a command

1. Add the executor in `internal/mac` (or reuse an existing one).
2. Register it in `internal/commands/register.go`.
3. Add the intent slot / utterance in the Alexa Developer Console.
4. Add tests; update `docs/security.md` and this README.
5. `make test && make vet`.

The Alexa layer never learns how a command runs, and the Mac layer never
learns about Alexa.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `HOST 0.0.0.0 is not allowed` | use a loopback address; the server refuses public binds by design |
| skill says "the remote endpoint could not be called" | tunnel down? check `cloudflared` logs; check the endpoint path is `/alexa` |
| 401 in server logs (`alexa request rejected`) | wrong `ALEXA_SKILL_ID`, or a stale signing cert (re-fetched automatically, cached 6 h) |
| file search says "not configured" | set `SEARCH_ROOTS` |
| `search_files` finds nothing | run `sudo mdutil -E /` once to index |
| Shortcut doesn't run | run it once manually to grant Automation permission |

## Future

An LLM layer is planned after the MVP stabilizes (PRD §27): it will emit
structured actions validated by the same policy layer and registry — never
direct shell access. Natural-language file search (PRD §28) builds on the
same path.
