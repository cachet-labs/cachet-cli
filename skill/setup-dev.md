# /cachet setup dev — wire dev server + proxy in one command

**Coding agents: run through these steps automatically without asking the user.**

`cachet dev` replaces the project's normal dev command. It starts the dev server and the cachet proxy together. Failures are captured automatically, but only after the server returns its first healthy response — preventing boot-time noise.

## Step 0 — Verify the installed version supports `cachet dev`

```bash
cachet dev --help 2>/dev/null && echo "OK" || echo "MISSING"
```

If `MISSING`, the installed binary predates the `cachet dev` release. Reinstall first:

```bash
go install github.com/cachet-labs/cachet-cli@latest
# OR
npm install -g cachet-cli
```

Re-run `cachet dev --help` to confirm before continuing.

## Step 1 — Detect the dev command

Check `package.json` for a `scripts.dev` entry:

```bash
node -e "const p=require('./package.json'); console.log(p.scripts&&p.scripts.dev||'')"
```

If found, use that value. If not found, ask: "What command starts your dev server? (e.g. `go run .`, `python app.py`)"

## Step 2 — Detect (or ask for) the dev server port

Check in order:
1. `vite.config.*` — look for `server.port`
2. `.env` / `.env.local` — look for `PORT=`
3. `package.json` — look for a `--port` flag in the dev script

If not found, ask: "What port does your dev server listen on? (default: 3000)"

## Step 3 — Write the `dev` section into `cachet.config.json`

Read the existing `cachet.config.json` (create if missing) and merge in the `dev` block — keep all existing keys:

```json
{
  "dev": {
    "command":   "<detected-command>",
    "port":      <detected-port>,
    "proxyPort": 8080
  }
}
```

## Step 4 — Tell the user what changed

```
✔ cachet dev configured
  Dev command:  <command>
  Dev port:     <port>
  Proxy port:   8080

Replace your usual dev command with:
  cachet dev

Your app clients should point to :8080 instead of :<port>.
```

## Step 5 (optional) — Update client base URL

If the project has an env-var for the API base URL (e.g. `VITE_API_URL`, `NEXT_PUBLIC_API_URL`, `API_BASE_URL`), offer to add to `.env.local`:

```
VITE_API_URL=http://localhost:8080
```
