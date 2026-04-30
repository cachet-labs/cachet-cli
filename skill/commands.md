# /cachet explain — command reference

## capture

Store a failure locally in `.cachet/recent/<id>.json`.

```bash
cachet capture --url POST:/pay --status 500 --error timeout
cachet capture --url POST:https://api.example.com/pay --status 500 --error timeout
cachet capture --url GET:/users/123 --status 404 --error not_found --body '{"id":123}'
```

`--url` accepts `METHOD:PATH` or `METHOD:FULL_URL`. Stdin JSON mode:

```bash
cat failure.json | cachet capture
# shape: { "request": { "method", "url", "headers", "body" }, "response": { "status" }, "error": { "type", "message" } }
```

Secrets are stripped before anything is written to disk.

---

## ask

Build a TOON-encoded LLM prompt and return a diagnosis.

```bash
cachet ask <id>
cachet ask --latest
cachet ask <id> --clipboard
```

Prompt includes: sanitized failure + up to 3 past cases (confidence ≥ 0.5) with the same fingerprint.

No LLM configured → prompt printed to stdout:
```bash
cachet ask --latest | pbcopy
cachet ask --latest > prompt.txt
```

---

## verify

Replay + git diff → LLM extracts root cause → stores Case globally.

```bash
cachet verify <id>                          # replay + git diff HEAD~1
cachet verify <id> --no-replay              # diff only (non-idempotent endpoints)
cachet verify <id> --diff HEAD~2            # custom git base
cachet verify <id> --base-url https://api.example.com
```

Requires a real LLM. Writes to `~/.cachet/cases/<id>.json` and updates `~/.cachet/index.json`.

---

## dev

Start dev server + capturing proxy as a single process.

```bash
cachet dev                              # uses cachet.config.json "dev" section
cachet dev --command "bun run dev"      # one-off override
cachet dev --port 4000 --proxy-port 8080
```

Captures are held until the dev server returns its first healthy response (status < min-status), preventing boot-time noise.

Config:
```json
{ "dev": { "command": "bun run dev", "port": 3000, "proxyPort": 8080, "minStatus": 400 } }
```

---

## proxy

Reverse proxy that auto-captures failing requests.

```bash
cachet proxy --port 8080 --target http://localhost:3000
cachet proxy --port 8080 --target http://localhost:3000 --min-status 500
```

---

## watch

Poll ngrok inspection API and auto-capture failures.

```bash
cachet watch --ngrok
cachet watch --ngrok --port 4041
cachet watch --ngrok --min-status 500
```

---

## latest

Print the most recent failure ID — useful in shell pipelines.

```bash
cachet latest
cachet ask $(cachet latest)
cachet verify $(cachet latest)
```

---

## replay

Re-execute a captured request against a live server.

```bash
cachet replay <id>
cachet replay <id> --base-url https://api.example.com
```

If the stored URL is a relative path, `--base-url` is required. Redacted headers are skipped.

---

## cases

List all resolved cases in `~/.cachet/cases/`.

```bash
cachet cases
cachet cases --filter category=timeout
cachet cases --filter confidence=0.8
```

---

## show

Inspect a single resolved case.

```bash
cachet show <case-id>
```

Prints: ID, fingerprint, category, confidence, created timestamp, root cause, fix.
