---
name: cachet
description: Interactive guide for cachet-cli — capture API failures, diagnose with AI, set up proxy/watch modes, and manage resolved cases with memory
---

# /cachet

Interactive usage guide and setup wizard for **cachet-cli** — the AI-powered API failure debugger with memory.

## Usage

```
/cachet                      # quick start + overview
/cachet setup                # first-time config wizard
/cachet setup dev            # wire cachet dev as the project's dev command (agents: do this automatically)
/cachet setup proxy          # set up proxy mode step-by-step
/cachet setup watch          # set up ngrok watch mode
/cachet setup anthropic      # configure Anthropic (Claude) as the LLM provider
/cachet setup openai         # configure OpenAI as the LLM provider
/cachet workflow             # full capture → ask → verify loop diagram
/cachet explain <command>    # deep-dive: capture, ask, verify, dev, proxy, watch, replay, cases, show, latest
```

---

## What cachet is for

`cachet` sits between your app and an LLM. When an API call fails, you capture it. cachet fingerprints it, strips secrets, builds a high-quality debugging prompt encoded in **TOON** (Token-Oriented Object Notation) to minimize token usage, and sends it to your LLM. When you fix the bug, you verify it — cachet extracts a structured Case (root cause, fix, category, confidence) and stores it globally. Next time the same class of failure happens, the past fix is injected automatically.

**The loop:**
```
failure happens → cachet capture → cachet ask → fix the bug → cachet verify → case stored
                                                                      ↑
                                        injected into future "ask" prompts automatically
```

---

## What You Must Do When Invoked

Read the user's args and respond accordingly. If no args, print the Quick Start below, then offer to walk through setup.

---

## Quick Start (no args)

Show this to the user:

```
cachet-cli quick start
──────────────────────────────────────────────────────────
1. Install
   go install github.com/cachet-labs/cachet-cli@latest

2. Configure LLM (optional — works without config too)
   Create cachet.config.json in your project root:
   {
     "provider": "anthropic",
     "apiKey": "sk-ant-...",
     "model": "claude-sonnet-4-6"
   }

3. Capture a failure
   cachet capture --url POST:/pay --status 500 --error timeout

4. Diagnose it
   cachet ask --latest

5. Fix it, then verify
   cachet verify <failure-id>
──────────────────────────────────────────────────────────
```

Then ask: "Want me to walk you through setup, proxy mode, or a specific command?"

---

## /cachet setup (first-time config)

Guide the user through creating `cachet.config.json` in their project root.

### Step 1 — Check if config exists

```bash
ls cachet.config.json 2>/dev/null && echo "EXISTS" || echo "MISSING"
```

If it exists, read it and show the current settings. Ask if they want to update it.

### Step 2 — Choose a provider

Ask: "Which LLM provider do you want to use?"
- **anthropic** — Claude (Sonnet 4.6 recommended)
- **openai** — GPT-4o or any OpenAI-compatible endpoint
- **none** — no-config mode: cachet prints the prompt to stdout so you can paste it anywhere

### Step 3 — Write the config

**Anthropic:**
```json
{
  "provider": "anthropic",
  "apiKey": "sk-ant-YOUR_KEY_HERE",
  "model": "claude-sonnet-4-6",
  "temperature": 0.2
}
```

**OpenAI:**
```json
{
  "provider": "openai",
  "apiKey": "sk-YOUR_KEY_HERE",
  "model": "gpt-4o",
  "temperature": 0.2
}
```

Write the config with the user's actual key. Remind them to gitignore it:

```bash
echo "cachet.config.json" >> .gitignore
```

### Step 4 — Environment variable alternative

If the user doesn't want to store the key in a file:

```bash
export CACHET_PROVIDER=anthropic
export CACHET_API_KEY=sk-ant-...
export CACHET_MODEL=claude-sonnet-4-6
```

Env vars take precedence over `cachet.config.json`.

---

## /cachet setup proxy

Proxy mode runs a local reverse proxy that auto-captures every failing request ≥ a status threshold. Zero code changes needed.

### When to use it
- You want automatic capture without instrumenting your app
- You're running a local dev server and want to intercept all failures
- You want to capture failures from any client (browser, mobile, curl)

### Step 1 — Verify your app is running

Ask the user: "What port is your app listening on?" (e.g. 3000)

### Step 2 — Start the proxy

```bash
cachet proxy --port 8080 --target http://localhost:3000
```

This starts cachet on `:8080`. All traffic is forwarded to `localhost:3000`. Any response with status ≥ 400 is automatically captured.

**Only capture 5xx errors:**
```bash
cachet proxy --port 8080 --target http://localhost:3000 --min-status 500
```

### Step 3 — Point your client at the proxy

Change your app's API base URL from `http://localhost:3000` to `http://localhost:8080`. Or set an env var:

```bash
API_BASE_URL=http://localhost:8080 npm run dev
```

### Step 4 — Trigger a failure

Make a request that fails. cachet prints:

```
✔  captured  POST /pay  →  500
   ID           f_abc123...
   Fingerprint  POST:/pay:500:timeout
   → cachet ask f_abc123...
```

### Step 5 — Diagnose

```bash
cachet ask --latest
# or
cachet ask f_abc123...
```

### Proxy architecture

```
Client → :8080 (cachet proxy) → :3000 (your app)
                    ↓ on status ≥ min-status
              .cachet/recent/<id>.json
```

The proxy uses a custom `http.RoundTripper` that intercepts responses before returning them to the client. The client sees the real response unchanged — cachet only taps the stream.

---

## /cachet setup watch

Watch mode polls an active dev tunnel and auto-captures failures without changing any URLs.

### When to use it
- You're using ngrok to expose a local server
- You can't change the base URL (mobile app, webhook consumer, etc.)
- You want zero-touch capture

### ngrok setup

```bash
# Start your app
npm run dev  # or whatever starts :3000

# Start ngrok in another terminal
ngrok http 3000

# Start cachet watch in a third terminal
cachet watch --ngrok
```

cachet polls `http://localhost:4040` (ngrok's local inspection API) every 2 seconds. Any request with response status ≥ 400 is captured automatically.

**Non-default inspection port:**
```bash
cachet watch --ngrok --port 4041
```

**Only capture 5xx:**
```bash
cachet watch --ngrok --min-status 500
```

### How it works

ngrok exposes a local REST API at `http://localhost:4040/api/requests/http` that lists all recent tunneled requests with full request/response data. cachet polls this, deduplicates by ngrok request ID, and ingests new failures through the same redact → fingerprint → store pipeline as `capture`.

---

## /cachet setup dev

**Coding agents: run through these steps automatically without asking the user.**

`cachet dev` replaces the project's normal dev command. It starts the dev server and the cachet proxy together as a single process. Failures are captured automatically, but only after the server handles its first healthy response — preventing boot-time noise.

### Step 1 — Detect the dev command

Check `package.json` for a `scripts.dev` entry:

```bash
node -e "const p=require('./package.json'); console.log(p.scripts&&p.scripts.dev||'')"
```

If found, use that value as the command. If not found, ask the user: "What command starts your dev server? (e.g. `go run .`, `python app.py`)"

### Step 2 — Detect (or ask for) the dev server port

Check common config files in order:
1. `vite.config.*` — look for `server.port`
2. `.env` / `.env.local` — look for `PORT=`
3. `package.json` — look for a `--port` flag in the dev script

If not found, ask: "What port does your dev server listen on? (default: 3000)"

### Step 3 — Write the `dev` section into `cachet.config.json`

Read the existing `cachet.config.json` (create it if missing) and merge in the `dev` block:

```json
{
  "dev": {
    "command":   "<detected-command>",
    "port":      <detected-port>,
    "proxyPort": 8080
  }
}
```

Keep all existing keys. Only add/update the `"dev"` key.

### Step 4 — Tell the user what changed

Print:
```
✔ cachet dev configured
  Dev command:  <command>
  Dev port:     <port>
  Proxy port:   8080

Replace your usual dev command with:
  cachet dev

Your app clients should point to :8080 instead of :<port>.
```

### Step 5 (optional) — Update client base URL

If the project uses an env-var for the API base URL (e.g. `VITE_API_URL`, `NEXT_PUBLIC_API_URL`, `API_BASE_URL`), offer to add it to `.env.local`:

```
VITE_API_URL=http://localhost:8080
```

---

## /cachet explain dev

`cachet dev` starts your dev server and cachet proxy as a single process.

```bash
cachet dev                              # uses cachet.config.json "dev" section
cachet dev --command "bun run dev"      # one-off override
cachet dev --port 4000 --proxy-port 8080
```

**Config (`cachet.config.json`):**
```json
{
  "dev": {
    "command":   "bun run dev",
    "port":      3000,
    "proxyPort": 8080,
    "minStatus": 400
  }
}
```

**Flags:**
```
--command      dev server shell command (overrides config)
--port         dev server port (overrides config)
--proxy-port   cachet proxy port (overrides config, default 8080)
--min-status   lowest status code to capture (overrides config, default 400)
```

**How boot-time noise is suppressed:**
The proxy holds all captures until the dev server returns its first response with status < min-status. Connection errors during startup are silently dropped.

---

## /cachet workflow

The full loop with all commands:

```
┌─────────────────────────────────────────────────────────┐
│  Capture                                                │
│  cachet capture --url POST:/pay --status 500 \          │
│                 --error timeout                         │
│                                                         │
│  Or pipe JSON:                                          │
│  cat failure.json | cachet capture                      │
│                                                         │
│  Or auto: cachet proxy / cachet watch                   │
└──────────────────┬──────────────────────────────────────┘
                   │ .cachet/recent/<id>.json
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Diagnose                                               │
│  cachet ask <id>              # send to LLM             │
│  cachet ask --latest          # most recent failure     │
│  cachet ask <id> --clipboard  # copy response           │
│                                                         │
│  No LLM? Prompt goes to stdout:                         │
│  cachet ask <id> | pbcopy                               │
└──────────────────┬──────────────────────────────────────┘
                   │ fix the bug
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Verify (stores a Case with memory)                     │
│  cachet verify <id>               # replay + git diff   │
│  cachet verify <id> --no-replay   # diff only           │
│  cachet verify <id> --diff HEAD~2 # custom git base     │
└──────────────────┬──────────────────────────────────────┘
                   │ ~/.cachet/cases/<case-id>.json
                   │ ~/.cachet/index.json updated
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Memory — auto-injected on next ask                     │
│  cachet cases                     # list all cases      │
│  cachet cases --filter category=timeout                 │
│  cachet show <case-id>            # inspect one case    │
└─────────────────────────────────────────────────────────┘
```

---

## /cachet explain capture

`cachet capture` stores a failure locally in `.cachet/recent/<id>.json`.

**Flag mode:**
```bash
cachet capture --url POST:/pay --status 500 --error timeout
cachet capture --url POST:https://api.example.com/pay --status 500 --error timeout
cachet capture --url GET:/users/123 --status 404 --error not_found --body '{"id":123}'
```

`--url` accepts `METHOD:PATH` or `METHOD:FULL_URL`. Full URLs are normalized to path-only for storage.

**Stdin JSON mode:**
```bash
cat failure.json | cachet capture

# failure.json shape:
{
  "request": {
    "method": "POST",
    "url": "/pay",
    "headers": { "Content-Type": "application/json", "Authorization": "Bearer sk-..." },
    "body": "{\"amount\": 100}"
  },
  "response": { "status": 500 },
  "error": { "type": "timeout", "message": "upstream timed out after 30s" }
}
```

Secrets in `Authorization`, `Cookie`, `X-Api-Key`, `X-Auth-Token` are **stripped automatically** before anything is written to disk. Bearer tokens, JWTs, emails, and AWS key IDs are masked to `[REDACTED]`.

---

## /cachet explain ask

`cachet ask` builds a structured LLM prompt and returns a diagnosis.

```bash
cachet ask f_550e8400-...              # diagnose by ID
cachet ask --latest                    # most recently captured failure
cachet ask f_550e8400-... --clipboard  # copy response to clipboard
```

**What the prompt includes:**
1. The sanitized failure (method, path, status, error, request body)
2. Up to 3 past resolved cases with the same fingerprint (confidence ≥ 0.5)
3. Instructions to produce root cause, likely fix, and category

**No LLM configured?** The prompt is printed to stdout — pipe it anywhere:
```bash
cachet ask --latest | pbcopy          # paste into Claude.ai
cachet ask --latest > prompt.txt      # save to file
```

**LLM selection logic:**
- `provider=anthropic` + `apiKey` set → Anthropic SDK
- `provider=openai` + `apiKey` set → OpenAI HTTP adapter
- anything else → stdout adapter (no error, pipe-ready)

---

## /cachet explain verify

`cachet verify` closes the loop: replays the request, diffs your recent code change, sends a resolver prompt to the LLM, and stores a structured **Case** globally.

```bash
cachet verify f_550e8400-...                  # replay + git diff HEAD~1
cachet verify f_550e8400-... --no-replay      # skip replay (safe for non-idempotent endpoints)
cachet verify f_550e8400-... --diff HEAD~2    # diff against two commits back
cachet verify f_550e8400-... --base-url https://api.example.com
```

**What it does:**
1. Optionally replays the request to confirm it now returns < 500
2. Runs `git diff <ref>` to capture your fix
3. Sends a resolver prompt: failure + diff → LLM extracts root cause, fix, category, confidence
4. Writes `~/.cachet/cases/<case-id>.json`
5. Updates `~/.cachet/index.json` (fingerprint → case IDs)

**Requires a real LLM** — verify does not work in stdout/no-config mode.

---

## /cachet explain proxy

See `/cachet setup proxy` for a full walkthrough.

**Flags:**
```
--port        port to listen on (default 8080)
--target      upstream URL to proxy to (required, e.g. http://localhost:3000)
--min-status  lowest status code to capture (default 400)
```

---

## /cachet explain watch

See `/cachet setup watch` for a full walkthrough.

**Flags:**
```
--ngrok       watch an ngrok tunnel (required flag)
--port        ngrok inspection API port (default 4040)
--min-status  lowest status code to capture (default 400)
```

---

## /cachet explain replay

Re-execute a captured request against a live server.

```bash
cachet replay f_550e8400-...
cachet replay f_550e8400-... --base-url https://api.example.com
```

If the stored URL is a relative path, `--base-url` is required. Redacted headers are skipped automatically.

---

## /cachet explain cases

List all resolved cases stored in `~/.cachet/cases/`.

```bash
cachet cases                            # all cases
cachet cases --filter category=timeout  # by error category
cachet cases --filter confidence=0.8    # confidence ≥ 0.8
```

---

## /cachet explain show

Inspect a single resolved case.

```bash
cachet show c_abc123...
```

Prints ID, fingerprint, category, confidence, created timestamp, root cause, and fix.

---

## /cachet explain latest

Print the most recently captured failure ID — useful in pipelines.

```bash
cachet latest
cachet ask $(cachet latest)
cachet verify $(cachet latest)
```

---

## Storage layout

```
<project>/
└── .cachet/
    └── recent/
        └── f_<uuid>.json    ← ephemeral, per-project

~/.cachet/
├── cases/
│   └── c_<uuid>.json        ← persistent resolved cases
└── index.json               ← fingerprint → [case_id, ...] map
```

---

## Fingerprint format

```
METHOD:NORMALIZED_ROUTE:STATUS:ERROR_TYPE

Examples:
  POST:/pay:500:timeout
  GET:/users/:id:404:not_found
  PUT:/orders/:id/items/:id:422:validation_error
```

UUIDs, numeric segments, and long hex strings are normalized to `:id`.

---

## Redaction defaults (always on, no config needed)

**Headers stripped:** `Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `X-Auth-Token`

**Values masked to `[REDACTED]`:** Bearer tokens, JWTs (`eyJ...`), email addresses, AWS key IDs (`AKIA...`)

**Add custom patterns** in `cachet.config.json`:
```json
{
  "redact": {
    "headers": ["X-Internal-Token"],
    "patterns": ["my-secret-[a-z0-9]+"]
  }
}
```

---

## Token efficiency — TOON encoding

All LLM prompts (ask and verify) are encoded in **TOON** (Token-Oriented Object Notation) — a compact, lossless alternative to JSON designed for LLM input. Data is stored as JSON on disk; TOON is only used at the LLM boundary.

TOON combines YAML-style indentation for nested objects with CSV-style tabular rows for uniform arrays:

```
failure:
  fingerprint: POST:/pay:500:timeout
  request:
    method: POST
    url: /pay
    body: {"amount":100}
  response:
    status: 500
  error:
    type: timeout
    message: payment service did not respond within 30s

similar_cases[3]{fingerprint,rootCause,fix,category,confidence}:
  POST:/pay:500:timeout,DB pool exhausted,Set MAX_CONNECTIONS=20,timeout,0.91
  POST:/pay:500:timeout,Redis timeout,Added circuit breaker,timeout,0.78
  POST:/pay:500:timeout,Stripe SDK timeout too low,Set stripe.Timeout=45s,upstream,0.65
```

The tabular format for past cases is the biggest win: 3 cases go from 12+ lines (with repeated `Fingerprint:`, `Root Cause:`, `Fix:` keys) to 4 lines.

See: [github.com/toon-format/toon](https://github.com/toon-format/toon)

---

## Common troubleshooting

**"no failures captured yet"** — Run `cachet capture --url GET:/test --status 500 --error test` to create one.

**"verify requires an LLM"** — Add `provider` and `apiKey` to `cachet.config.json` or set `CACHET_PROVIDER` / `CACHET_API_KEY`.

**Proxy not capturing** — Confirm `--target` is a full URL with scheme (`http://localhost:3000`). Check `--min-status` (default 400, not 500).

**Watch not capturing** — Start ngrok before `cachet watch`. Confirm inspection API is up: `curl http://localhost:4040/api/requests/http`.

**"URL is a relative path — supply --base-url"** — Use `--base-url https://your-api.com` with `replay` or `verify`.

**Prompt printed instead of LLM response** — Intentional no-config mode. Set `provider` + `apiKey` to get LLM responses.
