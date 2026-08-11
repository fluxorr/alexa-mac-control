# Cloudflare Tunnel

The Alexa skill needs a public HTTPS endpoint, but the controller must never
be exposed on the open internet (PRD §20). A Cloudflare Tunnel gives Alexa a
public URL that forwards to `127.0.0.1:2014` without opening router ports,
without a public IP, and without the controller learning anything about
Cloudflare.

## Prerequisites

- A Cloudflare account (free tier is enough)
- `cloudflared` installed:

  ```sh
  brew install cloudflared
  ```

## Option A — quick tunnel (development)

For a throwaway URL good for testing:

```sh
cloudflared tunnel --url http://127.0.0.1:2014
```

Copy the printed `https://<random>.trycloudflare.com` URL and use it as the
skill endpoint: `https://<random>.trycloudflare.com/alexa`.

Downside: the URL changes on every restart, so it must be re-entered in the
Alexa Developer Console each time.

## Option B — named tunnel (stable)

A named tunnel keeps the same URL across restarts:

```sh
cloudflared tunnel login
cloudflared tunnel create alexa-mac
```

Configure `~/.cloudflared/config.yml`:

```yaml
tunnel: alexa-mac
credentials-file: /Users/<you>/.cloudflared/<tunnel-id>.json

ingress:
  - hostname: alexa.<your-domain>.com
    service: http://127.0.0.1:2014
  - service: http_status:404
```

Add the DNS record (hostname → tunnel) with:

```sh
cloudflared tunnel route dns alexa-mac alexa.<your-domain>.com
```

Run it:

```sh
cloudflared tunnel run alexa-mac
```

The skill endpoint becomes `https://alexa.<your-domain>.com/alexa`.

## Security notes

- The tunnel forwards to `127.0.0.1` only; nothing listens on a public
  interface, ever. The server itself rejects non-loopback binds at startup.
- Alexa request verification (signature, timestamp, skill ID) happens inside
  the Go server, so a leaked tunnel URL is useless to an attacker: requests
  without a valid Alexa signature are rejected with 401.
- The local development endpoint `POST /api/command` is **not** reachable
  through the tunnel path (`/alexa` only), and it binds to loopback anyway.
- Do not expose the tunnel to the public internet beyond Alexa; if you do,
  the verification layer is the only thing standing between the world and
  your Mac — keep it intact.
