# ◆ cachet

**Turn runtime API failures into structured AI-debugging context with memory.**

`cachet` sits between your app (where errors happen) and an LLM, builds high-quality diagnostic prompts, and learns from past fixes by storing resolved cases globally — so the same failure is never diagnosed from scratch twice.

```
cachet capture --url POST:/pay --status 500 --error timeout
cachet ask <failure-id>
cachet verify <failure-id>
cachet cases
```

---

## Install

```bash
npm install -g cachet-cli
```

Or download a binary directly from [GitHub Releases](https://github.com/cachet-labs/cachet-cli/releases).

---

## Quick start

### 1. Configure (one-time)

```bash
cachet config init
```

Or set environment variables:

```bash
export CACHET_PROVIDER=anthropic
export CACHET_API_KEY=sk-ant-...
```

Supports **Anthropic** (`claude-sonnet-4-6`) and **OpenAI** (`gpt-4o`).

### 2. Capture a failure

From flags:
```bash
cachet capture --url POST:/pay --status 500 --error timeout --body '{"amount":100}'
```

From stdin JSON (pipe from your app or log):
```bash
cat failure.json | cachet capture
```

`capture` never makes network calls. It redacts secrets, fingerprints the route, and stores the failure locally in `.cachet/recent/`.

### 3. Ask for a diagnosis

```bash
cachet ask <failure-id>
```

- With an LLM configured: sends a structured prompt and displays the diagnosis.
- Without config: prints the prompt to stdout — pipe it anywhere:

```bash
cachet ask <id> | pbcopy          # macOS clipboard
cachet ask <id> --clipboard       # cross-platform
cachet ask <id> > prompt.txt
```

### 4. Verify a fix and store the case

After you fix the bug:

```bash
cachet verify <failure-id>
```

`verify` replays the request, captures `git diff HEAD~1`, and sends both to the LLM resolver. The structured result (root cause, fix, category, confidence) is stored globally in `~/.cachet/cases/` and indexed by fingerprint.

Future calls to `cachet ask` for the same endpoint + error pattern automatically inject matching cases into the prompt.

### 5. Browse your knowledge base

```bash
cachet cases                          # list all
cachet cases --filter category=timeout
cachet cases --filter confidence=0.8

cachet show <case-id>                 # inspect one
```

---

## All commands

| Command | Description |
|---|---|
| `cachet capture` | Capture a failure from flags or stdin JSON |
| `cachet ask <id>` | Diagnose with AI (or print prompt if unconfigured) |
| `cachet verify <id>` | Replay + diff → resolve → store case |
| `cachet replay <id>` | Re-execute the stored request |
| `cachet cases` | List all resolved cases |
| `cachet show <id>` | Inspect a single case |
| `cachet config init` | Interactive setup wizard |

---

## Redaction

Secrets are stripped **before** any prompt is built, LLM call is made, or data is written to disk.

Defaults (always applied):
- Headers: `Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `X-Auth-Token`
- Values: Bearer tokens, JWTs, email addresses, AWS key IDs

Add custom patterns in `cachet.config.json`:
```json
{
  "redact": {
    "headers": ["X-Internal-Token"],
    "patterns": ["secret_[a-z0-9]+"]
  }
}
```

---

## Storage

| What | Where |
|---|---|
| Captured failures | `./.cachet/recent/<id>.json` |
| Resolved cases | `~/.cachet/cases/<id>.json` |
| Fingerprint index | `~/.cachet/index.json` |
| Config | `./cachet.config.json` (gitignored) |

---

## Shell completion

```bash
cachet completion bash   >> ~/.bashrc
cachet completion zsh    >> ~/.zshrc
cachet completion fish   > ~/.config/fish/completions/cachet.fish
```

---

## License

MIT
