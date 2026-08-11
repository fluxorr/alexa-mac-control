# Security Model

This project's one job is to let Alexa drive the Mac **without ever opening
a hole to arbitrary code execution** (PRD §5). Every layer below is enforced
in code and covered by tests.

## Layered defenses

```
Internet
  │
  ▼
Cloudflare Tunnel        → loopback-only; no ports opened, no public IP
  │
  ▼
Alexa verification       → certificate URL, chain trust, signature, timestamp
  │
  ▼
Skill ID check           → only the configured application is accepted
  │
  ▼
Intent parsing           → fixed intents; free text only in Query slots
  │
  ▼
Command registry         → allowlist only; unknown names rejected
  │
  ▼
Argument validation      → query bounded to 256 chars
  │
  ▼
Mac executor             → explicit argv, never a shell, no user input
```

## The allowlist rule

There is no `POST /execute`, no command string endpoint, no `sh -c`. Every
action is a named `Command` registered at startup (PRD §5):

```
open_spotify open_vscode open_terminal open_safari open_backend
search_web search_files system_status lock sleep coding_mode
```

A request can only name one of these. Anything else gets a 404/unknown-command
error. A future LLM (PRD §27) will produce structured actions that pass
through the same registry — it will never receive shell access.

## Alexa request verification (PRD §18)

For every request to `/alexa` the server checks, in order:

1. `SignatureCertChainUrl` is `https://s3.amazonaws.com/echo.api/...`
   (scheme, host, path and credentials are all validated).
2. The certificate chain downloads from that URL (cached for 6h), parses,
   and verifies against the system trust store.
3. The `Signature` header is a valid RSA-SHA256 (or ECDSA-SHA256) signature
   over the exact raw request body, using the certificate's public key.
4. The request timestamp is within 150 seconds of the server clock
   (replay protection).
5. The application ID in the request matches `ALEXA_SKILL_ID`.

Failures are rejected with 401 and never reveal internals.

## Network posture

- The server rejects any bind other than a loopback address at startup
  (`HOST=0.0.0.0` is a hard error).
- `/api/command` is a local development convenience (PRD §24). It is never
  routed through the tunnel — only `/alexa` is public.
- Request bodies are capped at 1 MiB.

## What is logged

Structured logs record request IDs, method, path, status, duration, intent,
command names and outcomes. **Never logged:** tokens, secrets, signatures,
raw request bodies, command strings, file contents.

## What is never done

- `sudo`, `rm -rf`, `shutdown`, `reboot`, firewall or system modification:
  never executed by the server (restart is deliberately not implemented in
  v1; it requires a confirmation flow, PRD §12-§13).
- Credentials stored in source. `.env` is gitignored; `.env.example` is
  committed and documented.
- Weakened verification: there is no mode in which a public request skips
  signature checks. The only unverified mode is the unit tests, and
  `ALEXA_SKILL_ID` unset disables the **endpoint** entirely.

## Testing posture

Tests never touch the real system (PRD §25): they use fake executors, and
the verification suite signs requests with a locally generated test
certificate chain. Nothing in `go test ./...` can lock, sleep, open apps,
or modify files on the machine running it.
