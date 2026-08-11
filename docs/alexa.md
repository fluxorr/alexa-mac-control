# Alexa Skill Setup

## 1. Create the skill

1. Go to the [Alexa Developer Console](https://developer.amazon.com/alexa/console/ask).
2. **Create Skill** → name it `Mac` → choose **Custom** model → **Start from scratch**.
3. Choose any default language; en-US works with the utterances below.
4. For "Choose a method to host your skill's backend resources" pick **Provision your own** (you do not need the ASK SDK; the Go server is the endpoint).

## 2. Invocation name

Set the invocation name to:

```
mac
```

So the user says: "Alexa, ask **Mac** to open Spotify."

## 3. Intents

Create exactly these intents:

| Intent | Sample utterances |
|---|---|
| `MacCommandIntent` | "open Spotify", "open VS Code", "open terminal", "open safari", "open the backend project", "lock", "sleep", "start coding mode" |
| `MacSearchIntent` | "search for {Query}", "search {Query}", "look up {Query}" |
| `MacFileSearchIntent` | "search my files for {Query}", "find files about {Query}" |
| `MacStatusIntent` | "system status", "status", "how is my Mac doing" |

Keep `AMAZON.HelpIntent`, `AMAZON.CancelIntent`, `AMAZON.StopIntent`, and
`AMAZON.FallbackIntent` (added by default).

`MacSearchIntent` and `MacFileSearchIntent` need one slot each: `Query` of
type `AMAZON.SearchQuery`. `MacCommandIntent` needs one slot: `Action` of
type `AMAZON.Custom` with the sample utterances above as slot values (or use
`AMAZON.Literal`).

## 4. Endpoint

1. Get your public HTTPS URL from the Cloudflare Tunnel (see
   [cloudflare.md](cloudflare.md)).
2. In the skill's **Endpoint** tab, set:
   - Service Endpoint Type: **HTTPS**
   - Default Region: `https://<your-tunnel>.trycloudflare.com/alexa`
   - SSL certificate type: **My development endpoint is a subdomain of a domain with a wildcard certificate from a certificate authority** (Cloudflare serves publicly trusted certificates).

## 5. Skill ID

1. From the **Build** tab, copy the **Skill ID** (bottom of the left pane,
   format `amzn1.ask.skill.xxxx`).
2. Set it on the Mac:

   ```sh
   export ALEXA_SKILL_ID=amzn1.ask.skill.xxxx
   ```

   The server rejects any request carrying a different skill ID.

## 6. Test

1. In the Developer Console **Test** tab, select **Development**.
2. Type or speak: `ask Mac to open Spotify`.
3. The skill calls the tunnel URL; the Go server verifies the signature and
   replies with speech text, e.g. "Opening Spotify."

## 7. Signature verification

The server verifies every request (see [security.md](security.md)):

- certificate URL is `https://s3.amazonaws.com/echo.api/...`
- certificate chains to a trusted root
- RSA/ECDSA signature over the exact raw body
- timestamp within 150 seconds (replay protection)
- skill ID matches `ALEXA_SKILL_ID`

No extra work is needed on the console side — Amazon sends the
`Signature` and `SignatureCertChainUrl` headers automatically for HTTPS
endpoints.
