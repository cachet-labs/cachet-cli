# /cachet setup proxy — local reverse proxy

Proxy mode auto-captures every failing request ≥ a status threshold. Zero code changes needed.

## When to use it
- Automatic capture without instrumenting your app
- Local dev server with any client (browser, mobile, curl)
- You can change the base URL your client calls

## Step 1 — Start the proxy

```bash
cachet proxy --port 8080 --target http://localhost:3000
```

Starts cachet on `:8080`. All traffic forwarded to `:3000`. Any response ≥ 400 is captured automatically.

Only 5xx:
```bash
cachet proxy --port 8080 --target http://localhost:3000 --min-status 500
```

## Step 2 — Point your client at the proxy

Change base URL from `http://localhost:3000` → `http://localhost:8080`, or:

```bash
API_BASE_URL=http://localhost:8080 npm run dev
```

## Step 3 — Trigger a failure

Make a request that fails. cachet prints:

```
✔  captured  POST /pay  →  500
   ID           f_abc123...
   Fingerprint  POST:/pay:500:timeout
   → cachet ask f_abc123...
```

## Step 4 — Diagnose

```bash
cachet ask --latest
```

## Architecture

```
Client → :8080 (cachet proxy) → :3000 (your app)
                    ↓ on status ≥ min-status
              .cachet/recent/<id>.json
```

The proxy uses a custom `http.RoundTripper` — client sees the real response unchanged.

## Flags

```
--port        port to listen on (default 8080)
--target      upstream URL (required, e.g. http://localhost:3000)
--min-status  lowest status to capture (default 400)
```
