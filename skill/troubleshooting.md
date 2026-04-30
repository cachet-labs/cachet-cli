# Troubleshooting + reference

## Common errors

**"no failures captured yet"**
```bash
cachet capture --url GET:/test --status 500 --error test
```

**"verify requires an LLM"**
Add `provider` and `apiKey` to `cachet.config.json` or set `CACHET_PROVIDER` / `CACHET_API_KEY`.

**"unknown command dev"**
Binary predates `cachet dev`. Reinstall:
```bash
go install github.com/cachet-labs/cachet-cli@latest
# OR
npm install -g cachet-cli
```

**Proxy not capturing**
- `--target` must be a full URL with scheme: `http://localhost:3000`
- Check `--min-status` (default 400, not 500)

**Watch not capturing**
- Start ngrok before `cachet watch`
- Confirm API: `curl http://localhost:4040/api/requests/http`

**"URL is a relative path — supply --base-url"**
Use `--base-url https://your-api.com` with `replay` or `verify`.

**Prompt printed instead of LLM response**
Intentional no-config mode. Set `provider` + `apiKey` to get LLM responses.

---

## Storage layout

```
<project>/
└── .cachet/recent/<id>.json       ← ephemeral, per-project

~/.cachet/
├── cases/<id>.json                ← persistent resolved cases
└── index.json                     ← fingerprint → [case_ids]
```

---

## Fingerprint format

```
METHOD:NORMALIZED_ROUTE:STATUS:ERROR_TYPE

POST:/pay:500:timeout
GET:/users/:id:404:not_found
PUT:/orders/:id/items/:id:422:validation_error
```

UUIDs, numeric segments, and long hex strings → `:id`.

---

## Redaction defaults (always on)

**Headers stripped:** `Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `X-Auth-Token`

**Values masked:** Bearer tokens, JWTs (`eyJ...`), email addresses, AWS key IDs (`AKIA...`)

Custom patterns in `cachet.config.json`:
```json
{ "redact": { "headers": ["X-Internal-Token"], "patterns": ["my-secret-[a-z0-9]+"] } }
```

---

## TOON encoding

LLM prompts are encoded in [TOON](https://github.com/toon-format/toon) (Token-Oriented Object Notation) — data stays JSON on disk, TOON is only used at the LLM input boundary.

Example (3 past cases):
```
similar_cases[3]{fingerprint,rootCause,fix,category,confidence}:
  POST:/pay:500:timeout,DB pool exhausted,Set MAX_CONNECTIONS=20,timeout,0.91
  POST:/pay:500:timeout,Redis timeout,Added circuit breaker,timeout,0.78
  POST:/pay:500:timeout,Stripe SDK timeout too low,Set stripe.Timeout=45s,upstream,0.65
```
