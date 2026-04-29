# Cachet CLI — Claude Context

## What This Is

`cachet` is an open-source Go CLI that turns runtime API failures into structured AI-debugging context with memory. It sits between your app (where errors happen) and an LLM (Claude, OpenAI, etc.), builds high-quality prompts, and learns from past fixes by storing resolved cases globally.

Full implementation plan and data models: `Plan.md`.

---

## Essential Commands

```bash
# Build
go build -o cachet .

# Run without installing
go run . capture --url POST:/pay --status 500 --error timeout

# Test
go test ./...
go test ./internal/core/... -v

# Vet + format (run before committing)
go vet ./...
gofmt -w .

# Install locally
go install .

# Add a dependency
go get github.com/some/dep
go mod tidy

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o cachet-linux .
GOOS=darwin GOARCH=arm64 go build -o cachet-darwin-arm64 .
```

---

## Project Structure

```
cachet-cli/
├── main.go                         # entrypoint — calls cmd.Execute()
├── go.mod / go.sum
│
├── cmd/                            # cobra command definitions (thin layer only)
│   ├── root.go                     # root cmd, persistent flags, .cachet/ init
│   ├── capture.go                  # cachet capture (stdin JSON or flags)
│   ├── ask.go                      # cachet ask <id> / --latest
│   ├── latest.go                   # cachet latest
│   ├── proxy.go                    # cachet proxy --port --target
│   ├── watch.go                    # cachet watch --ngrok
│   ├── verify.go                   # cachet verify <id>
│   ├── cases.go                    # cachet cases
│   ├── show.go                     # cachet show <case-id>
│   └── replay.go                   # cachet replay <id>
│
├── internal/
│   ├── core/
│   │   ├── fingerprint.go          # METHOD:NORMALIZED_ROUTE:STATUS:ERROR_TYPE
│   │   ├── formatter.go            # failure + past cases → LLM prompt string
│   │   ├── redact.go               # strip auth headers, mask tokens/emails
│   │   └── resolver.go             # git diff + LLM response → structured Case
│   │
│   ├── pipeline/
│   │   └── ingest.go               # shared redact→fingerprint→store sequence
│   │
│   ├── proxy/
│   │   └── proxy.go                # reverse proxy with capture transport
│   │
│   ├── watcher/
│   │   └── ngrok.go                # ngrok inspection API poller
│   │
│   ├── storage/
│   │   ├── local.go                # .cachet/recent/<id>.json + LatestID()
│   │   ├── global.go               # ~/.cachet/cases/<id>.json
│   │   └── index.go                # ~/.cachet/index.json (fingerprint→[case_ids])
│   │
│   └── llm/
│       ├── adapter.go              # LLMAdapter interface
│       ├── anthropic.go            # Anthropic adapter (official Go SDK)
│       ├── openai.go               # OpenAI adapter (raw HTTP, no SDK)
│       └── stdout.go               # no-config fallback: print prompt to stdout
│
└── pkg/
    └── config/
        └── config.go               # cachet.config.json loader via viper
```

**Rule:** `cmd/` files must stay thin — they parse flags, call `internal/` packages, and print results. No business logic in `cmd/`.

---

## Storage Locations

| What | Path |
|---|---|
| Captured failures (ephemeral) | `./.cachet/recent/<id>.json` |
| Resolved cases (persistent) | `~/.cachet/cases/<id>.json` |
| Fingerprint index | `~/.cachet/index.json` |
| User config | `./cachet.config.json` (gitignored) |

---

## Data Model Quick Reference

**Failure ID format:** `f_` prefix + UUID (e.g. `f_550e8400-e29b-41d4-a716-446655440000`)
**Case ID format:** `c_` prefix + UUID

**Fingerprint formula:**
```
METHOD:NORMALIZED_ROUTE:STATUS:ERROR_TYPE
```
Route normalization: UUID segments, numeric segments, and hash-like segments (>24 chars) all become `:id`.
Example: `GET /users/123 404 not_found` → `GET:/users/:id:404:not_found`

---

## Key Invariants — Never Violate These

1. **Redaction is always first.** `redact.Failure()` must be called before prompt building, before LLM send, and before writing any user-supplied body to disk.
2. **Storage is append-only.** Failures and cases are never mutated after write. New data always gets a new ID.
3. **`LLMAdapter` interface is the only LLM boundary.** No LLM SDK imports outside `internal/llm/`. All adapters implement `Ask(prompt string) (string, error)`.
4. **`capture` and `latest` never make network calls.** Only `ask`, `verify`, `replay`, `proxy`, and `watch` touch the network.
5. **No-config mode must always work.** If `cachet.config.json` is absent or has no provider, `ask` prints the prompt to stdout instead of erroring.

---

## Config Schema (`cachet.config.json`)

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

Env var overrides (take precedence over file):
- `CACHET_API_KEY` — overrides `apiKey`
- `CACHET_PROVIDER` — overrides `provider`
- `CACHET_MODEL` — overrides `model`

---

## Redaction Defaults (always applied, even without config)

**Headers stripped entirely:**
`Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`, `X-Auth-Token`

**Values masked to `[REDACTED]`:**
- Bearer tokens
- JWTs (`eyJ...`)
- Email addresses
- AWS access key IDs (`AKIA...`)

User patterns from `cachet.config.json` are appended, never replace, the defaults.

---

## LLM Adapter Selection Logic

```
if config.provider == "anthropic" && config.apiKey != ""  → anthropic adapter
if config.provider == "openai"    && config.apiKey != ""  → openai adapter
otherwise                                                  → stdout adapter
```

The stdout adapter prints the fully-built prompt and exits 0. It is not an error state.

---

## Error Handling Conventions

- Return `error` from all `internal/` functions; never `os.Exit` inside a library.
- `cmd/` files call `cobra.CheckErr()` or print and `os.Exit(1)` after getting an error from internal packages.
- Wrap errors with context: `fmt.Errorf("store failure: %w", err)`.
- Never swallow errors silently.

---

## Testing Conventions

- Unit tests live alongside their source files: `fingerprint_test.go` next to `fingerprint.go`.
- Use table-driven tests for `fingerprint` and `redact` — they have many input/output cases.
- No mocking of the filesystem — use `t.TempDir()` for storage tests.
- LLM adapters are tested with a stub that implements `LLMAdapter` and returns a fixed response string.
- Golden files for formatter output go in `internal/core/testdata/`.

---

## Go Style Conventions

- `gofmt` enforced — run `gofmt -w .` before committing.
- Exported types and functions get a single-line doc comment. Unexported ones only if the behavior is non-obvious.
- Prefer explicit error returns over panics everywhere except `main()` during setup.
- Keep `cmd/` files under ~100 lines each; extract helpers to `internal/` if they grow.
- IDs generated with `github.com/google/uuid` — always use `uuid.New().String()` and prepend the type prefix (`f_`, `c_`).

---

## Phase Status

| Phase | Scope | Status |
|---|---|---|
| 1 | capture, ask, cases, show + all internals | Complete |
| 2 | verify, replay, memory injection, OpenAI adapter | Complete |
| 3 | proxy, watch (ngrok), latest, pipeline refactor | Complete |
| 4 | shell completion, goreleaser, brew tap | Not started |

---

## What NOT to Do

- Do not add cloud sync, a web dashboard, or team-sharing features — these are explicitly out of scope.
- Do not import LLM SDKs outside `internal/llm/`.
- Do not add retries or circuit breakers to LLM calls in MVP — keep the adapter thin.
- Do not store raw user secrets in failures on disk — redaction must run before any write.
- Do not use `cobra.OnInitialize` for anything that does I/O beyond reading the config file.
- Do not add `--verbose` debug logging yet — defer to Phase 4.
