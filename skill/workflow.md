# /cachet workflow — full capture → ask → verify loop

```
┌─────────────────────────────────────────────────────────┐
│  Capture                                                │
│  cachet capture --url POST:/pay --status 500 \          │
│                 --error timeout                         │
│                                                         │
│  Or pipe JSON:  cat failure.json | cachet capture       │
│  Or auto:       cachet dev / cachet proxy / cachet watch │
└──────────────────┬──────────────────────────────────────┘
                   │ .cachet/recent/<id>.json
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Diagnose                                               │
│  cachet ask <id>              # send to LLM             │
│  cachet ask --latest          # most recent failure     │
│  cachet ask <id> --clipboard  # copy response           │
│                                                         │
│  No LLM? Prompt goes to stdout (TOON-encoded):          │
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

## Key facts

- `capture` and `latest` never make network calls
- `ask` prompts are TOON-encoded (compact, token-efficient)
- Past cases with the same fingerprint and confidence ≥ 0.5 are injected automatically into `ask` prompts
- `verify` requires a real LLM (not stdout mode)
- Storage is append-only — nothing is ever mutated after write
