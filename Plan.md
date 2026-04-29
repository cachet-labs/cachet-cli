# Cachet CLI — Implementation Plan

## Language & Runtime Decision: Go

**Choice: Go 1.22+**

Rationale over the alternatives:

| Concern | Go | TypeScript/Node | Python |
|---|---|---|---|
| Distribution | Single static binary — `brew install`, curl-pipe, no runtime deps | Requires Node.js on every machine | Requires Python + pip, venv mess |
| Startup | ~5ms | ~300–600ms (Node cold start) | ~200ms |
| CLI ecosystem | `cobra` (kubectl, docker, gh), `viper` | `commander`, `oclif` | `click`, `typer` |
| LLM integration | Official Anthropic Go SDK + raw HTTP for others | Most mature Anthropic/OpenAI SDKs | Excellent SDKs but distributing is painful |
| Open-source credibility | Go CLIs (fzf, gh, age, k9s) dominate the dev tool space | Strong but Node runtime is a friction point | Fine for scripts, not trusted as a system tool |
| Cross-platform | `GOOS=linux GOARCH=amd64 go build` | Needs `pkg` or similar | `pyinstaller` is brittle |

Go wins cleanly for a developer-facing CLI that needs to feel like a first-class tool, not a Node script wrapper.

**Key Go dependencies:**
- `github.com/spf13/cobra` — command routing
- `github.com/spf13/viper` — config loading
- `github.com/anthropics/anthropic-sdk-go` — Anthropic LLM adapter
- `github.com/google/uuid` — failure/case IDs
- `github.com/fatih/color` — terminal output
- Standard library only for everything else (HTTP, JSON, file I/O, exec)

---

## Design Decisions (Ambiguities Resolved)

**`capture` input shape:** Both stdin JSON and structured flags.
- `cachet capture < failure.json` — pipe a raw JSON failure blob
- `cachet capture --url POST:/pay --status 500 --error "timeout" --body '{...}'` — explicit flags
- Auto-detection: if stdin is a TTY, require flags; otherwise read JSON from stdin

**`verify` mechanism:** Auto-replay the captured request + extract `git diff`.
- `cachet verify <id>` replays the stored request, checks for non-5xx response, then runs `git diff HEAD~1` to collect the fix diff and sends both to the LLM resolver.
- If replay is not desired (read-only endpoint), `--no-replay` flag skips it and uses diff only.

**Default output (no LLM configured):** Structured prompt printed to stdout.
- Pipe-friendly by design: `cachet ask <id> | pbcopy`
- `--clipboard` flag for convenience copy (uses `pbcopy`/`xclip`/`clip` per OS)

**Rename from spec:** All `trace`/`.trace` references become `cachet`/`.cachet`.

---

## Project Structure

```
cachet-cli/
├── main.go
├── go.mod
├── go.sum
├── LICENSE
├── Plan.md
│
├── cmd/                        # cobra command definitions
│   ├── root.go                 # root command, persistent flags, config init
│   ├── capture.go              # cachet capture
│   ├── ask.go                  # cachet ask <id>
│   ├── verify.go               # cachet verify <id>
│   ├── cases.go                # cachet cases
│   ├── show.go                 # cachet show <case-id>
│   └── replay.go               # cachet replay <id>
│
├── internal/
│   ├── core/
│   │   ├── fingerprint.go      # METHOD+ROUTE+STATUS+ERROR_TYPE → hash
│   │   ├── formatter.go        # failure + cases → LLM prompt string
│   │   ├── redact.go           # strip auth headers, mask tokens/emails
│   │   └── resolver.go         # git diff + LLM response → structured Case
│   │
│   ├── storage/
│   │   ├── local.go            # .cachet/recent/ — Failure read/write
│   │   ├── global.go           # ~/.cachet/cases/ — Case read/write
│   │   └── index.go            # ~/.cachet/index.json — fingerprint→[case_ids]
│   │
│   └── llm/
│       ├── adapter.go          # LLMAdapter interface
│       ├── anthropic.go        # Anthropic adapter (official SDK)
│       ├── openai.go           # OpenAI adapter (raw HTTP)
│       └── stdout.go           # no-config fallback: print prompt to stdout
│
└── pkg/
    └── config/
        └── config.go           # cachet.config.json loader via viper
```

---

## Data Models

### Failure (ephemeral — `.cachet/recent/<id>.json`)

```json
{
  "id": "f_01HXYZ",
  "captured_at": "2026-04-29T10:00:00Z",
  "request": {
    "url": "https://api.example.com/pay",
    "method": "POST",
    "headers": { "Content-Type": "application/json" },
    "body": "{\"amount\": 100}"
  },
  "response": {
    "status": 500,
    "headers": { "Content-Type": "application/json" },
    "body": "{\"error\": \"timeout\"}"
  },
  "error": {
    "type": "timeout",
    "message": "upstream service timed out after 30s",
    "stack": ""
  },
  "fingerprint": "POST:/pay:500:timeout"
}
```

### Case (persistent — `~/.cachet/cases/<id>.json`)

```json
{
  "id": "c_01HABC",
  "fingerprint": "POST:/pay:500:timeout",
  "root_cause": "Payment service upstream timeout due to missing connection pool limit",
  "fix": "Added MAX_CONNECTIONS=20 to payment service config and retry with backoff",
  "category": "timeout",
  "confidence": 0.92,
  "created_at": "2026-04-29T10:05:00Z"
}
```

### Index (`~/.cachet/index.json`)

```json
{
  "POST:/pay:500:timeout": ["c_01HABC", "c_01HDEF"],
  "GET:/users:404:not_found": ["c_01HGHI"]
}
```

### Config (`cachet.config.json` — project root, gitignored)

```json
{
  "provider": "anthropic",
  "apiKey": "sk-ant-...",
  "model": "claude-sonnet-4-6",
  "temperature": 0.2,
  "redact": {
    "headers": ["Authorization", "Cookie", "X-Api-Key"],
    "patterns": ["Bearer [^ ]+", "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+"]
  }
}
```

---

## Fingerprinting Logic

```
fingerprint = METHOD + ":" + NORMALIZED_ROUTE + ":" + STATUS + ":" + ERROR_TYPE
```

Route normalization:
- `/users/123/orders/456` → `/users/:id/orders/:id`
- `/pay` → `/pay`

Rules:
- UUID segments → `:id`
- Numeric-only segments → `:id`
- Hash-like segments (>24 chars, alphanumeric) → `:id`

Examples:
- `POST /pay 500 timeout` → `POST:/pay:500:timeout`
- `GET /users/123 404 not_found` → `GET:/users/:id:404:not_found`

---

## Redaction Rules (Shipped Defaults)

Headers always stripped:
- `Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `X-Auth-Token`

Value patterns masked to `[REDACTED]`:
- Bearer tokens: `Bearer [A-Za-z0-9\-._~+/]+=*`
- JWT format: `eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`
- Email addresses
- AWS keys: `AKIA[0-9A-Z]{16}`
- Credit card numbers (Luhn-detectable, optional)

User-defined patterns added via `cachet.config.json`.

---

## Prompt Template

```
You are debugging a runtime API failure. Analyze the failure below and provide a structured diagnosis.

== REQUEST ==
Method: {{method}}
URL: {{url}}
Body: {{body}}

== RESPONSE ==
Status: {{status}}
Body: {{response_body}}

== ERROR ==
Type: {{error_type}}
Message: {{error_message}}
{{#if stack}}Stack: {{stack}}{{/if}}

{{#if similar_cases}}
== SIMILAR PAST ISSUES ==
{{#each similar_cases}}
- Fingerprint: {{fingerprint}}
  Root Cause: {{root_cause}}
  Fix: {{fix}}
{{/each}}
{{/if}}

== TASK ==
1. Identify the root cause (1 sentence)
2. Suggest a concrete fix (1–3 sentences)
3. List edge cases or related failure modes to watch for
4. Assign a category: timeout | auth | not_found | rate_limit | validation | upstream | config | unknown
```

---

## Resolver Prompt (used in `verify`)

```
A bug was fixed. Given the failure context and the git diff below, generate a structured resolution.

== ORIGINAL FAILURE ==
Fingerprint: {{fingerprint}}
Error: {{error_message}}

== GIT DIFF ==
{{diff}}

== OUTPUT FORMAT (strict) ==
Root Cause: <one sentence>
Fix: <one sentence>
Category: <timeout|auth|not_found|rate_limit|validation|upstream|config|unknown>
Confidence: <0.0–1.0>
```

---

## Phase Breakdown

### Phase 1 — Core Loop (this session)

Goal: end-to-end working flow: capture → ask → cases → show

- [x] `go.mod` init, dependency resolution
- [x] `pkg/config` — load `cachet.config.json` via viper, env var overrides (`CACHET_API_KEY` etc.)
- [x] `internal/core/fingerprint.go` — route normalization + fingerprint generation
- [x] `internal/core/redact.go` — default rules + config-defined patterns
- [x] `internal/core/formatter.go` — build LLM prompt from Failure + similar Cases
- [x] `internal/storage/local.go` — write/read `.cachet/recent/<id>.json`
- [x] `internal/storage/global.go` — write/read `~/.cachet/cases/<id>.json`
- [x] `internal/storage/index.go` — maintain `~/.cachet/index.json`
- [x] `internal/llm/adapter.go` — `LLMAdapter` interface: `Ask(prompt string) (string, error)`
- [x] `internal/llm/stdout.go` — print prompt to stdout (no-config mode)
- [x] `internal/llm/anthropic.go` — Anthropic adapter via official Go SDK
- [x] `cmd/root.go` — cobra root, config loading, `.cachet/` dir init
- [x] `cmd/capture.go` — stdin JSON + flag modes, redact, fingerprint, store locally
- [x] `cmd/ask.go` — load failure, fetch similar cases, build prompt, call adapter, print response
- [x] `cmd/cases.go` — list all global cases, tabular output
- [x] `cmd/show.go` — pretty-print a single case
- [x] `main.go` — entrypoint

### Phase 2 — Memory + Resolution

- [x] `internal/core/resolver.go` — git diff → resolver prompt → structured Case parse
- [x] `cmd/verify.go` — replay request, check status, run git diff, call resolver, store Case + update index
- [x] `cmd/replay.go` — re-execute stored request, print response
- [x] Memory injection in `ask` — fetch top 2–3 cases by fingerprint from index
- [x] `internal/llm/openai.go` — OpenAI adapter (raw HTTP, no SDK dep)

### Phase 3 — Polish & Extensions

- [ ] `--clipboard` flag on `ask`
- [ ] `cachet config init` — interactive setup wizard
- [ ] `cachet cases --filter category=timeout` — filtered listing
- [ ] Confidence threshold filtering on memory injection
- [ ] Shell completion (`cachet completion bash/zsh/fish`)
- [ ] Integration test suite (golden files for redact, fingerprint, formatter)
- [ ] Homebrew tap (`cachet-labs/homebrew-cachet-cli`)

---

## Distribution & Release

> Implemented ahead of phases 1–2. Inert until the first real `v0.1.0` tag is pushed.

### npm distribution (`npm/` directory)

Postinstall-download approach: one npm package, binary fetched from GitHub Releases at install time.

```
npm install -g cachet-cli      # Windows, macOS, Linux
```

**Files:**
- `npm/package.json` — `name: cachet-cli`, `bin: { cachet: bin/cachet.js }`, `postinstall: node install.js`
- `npm/install.js` — detects platform/arch, downloads binary, verifies SHA256 against goreleaser's `checksums.txt`, extracts, chmods
- `npm/bin/cachet.js` — thin JS shim that `spawnSync`s the downloaded binary with `stdio: inherit`

**Supported platforms:** `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`

### goreleaser (`.goreleaser.yml`)

Builds 6 targets, produces flat archives (no wrapping directory):
- `cachet_{version}_{os}_{arch}.tar.gz` for Unix
- `cachet_{version}_{os}_{arch}.zip` for Windows
- `checksums.txt` (SHA256, consumed by `install.js`)

Requires `main.go` to declare `var version = "dev"` — goreleaser injects the real version via `-X main.version={{.Version}}`.

### Release pipeline (`.github/workflows/release.yml`)

Triggered by `git tag v*` + push. Two sequential jobs:
1. `goreleaser` — builds all targets, creates GitHub Release with assets
2. `npm-publish` — strips `v` prefix from tag, bumps `npm/package.json` version, publishes to npm

**Secrets required in GitHub repository settings:**
- `NPM_TOKEN` — npm access token with publish rights to `cachet-cli`
- `GITHUB_TOKEN` — automatically provided by Actions (no setup needed)

---

## CLI Surface (Final)

```
cachet capture                    # read failure JSON from stdin
cachet capture --url POST:/pay --status 500 --error timeout [--body '{}']
cachet ask <failure-id>           # build prompt + send to LLM (or print if no config)
cachet ask <failure-id> --clipboard
cachet verify <failure-id>        # replay + diff → resolve → store case
cachet verify <failure-id> --no-replay
cachet cases                      # list all stored cases
cachet show <case-id>             # inspect one case
cachet replay <failure-id>        # re-execute request, print response
```

---

## Key Invariants

1. **Redaction runs before anything leaves the process** — before prompt build, before LLM send, before storage of any body that came from user input.
2. **Storage is append-only** — failures and cases are never overwritten; new versions get new IDs.
3. **LLM is pluggable** — `LLMAdapter` interface means swapping providers requires zero changes outside `llm/`.
4. **No network calls without explicit user action** — `capture` is local-only. Only `ask`, `verify`, `replay` touch the network.
5. **Graceful no-config mode** — every command works without a `cachet.config.json`; LLM calls degrade to stdout print.
