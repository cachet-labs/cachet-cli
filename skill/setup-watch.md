# /cachet setup watch — ngrok tunnel watcher

Watch mode polls an active ngrok tunnel and auto-captures failures without changing any URLs.

## When to use it
- You're using ngrok to expose a local server
- You can't change the base URL (mobile app, webhook consumer, etc.)
- Zero-touch capture

## Setup

```bash
# Terminal 1 — start your app
npm run dev

# Terminal 2 — start ngrok
ngrok http 3000

# Terminal 3 — start cachet watch
cachet watch --ngrok
```

cachet polls `http://localhost:4040/api/requests/http` every 2 seconds. Any request with status ≥ 400 is captured automatically.

## Flags

```bash
cachet watch --ngrok                  # default inspection port 4040
cachet watch --ngrok --port 4041      # non-default port
cachet watch --ngrok --min-status 500 # only 5xx
```

## How it works

ngrok exposes a local REST API at `http://localhost:4040/api/requests/http` with full request/response data. cachet polls this, deduplicates by ngrok request ID, and runs each failure through the same redact → fingerprint → store pipeline as `capture`.

## Troubleshooting

**Watch not capturing** — Start ngrok before `cachet watch`. Confirm the inspection API is up:
```bash
curl http://localhost:4040/api/requests/http
```
