---
name: cachet
description: Interactive guide for cachet-cli — capture API failures, diagnose with AI, set up proxy/watch/dev modes, and manage resolved cases with memory
---

# /cachet

Interactive setup wizard and usage guide for **cachet-cli** — the AI-powered API failure debugger with memory.

## Usage

```
/cachet                    # quick start
/cachet setup              # configure LLM provider
/cachet setup dev          # wire cachet dev as your dev command (agents: auto-run this)
/cachet setup proxy        # set up reverse proxy mode
/cachet setup watch        # set up ngrok watch mode
/cachet setup anthropic    # configure Anthropic / Claude
/cachet setup openai       # configure OpenAI
/cachet workflow           # full capture → ask → verify diagram
/cachet explain <command>  # deep-dive any command
```

## Quick Start (no args)

Show this, then ask what they need help with:

```
cachet-cli quick start
──────────────────────────────────────────────────────────
1. Install
   go install github.com/cachet-labs/cachet-cli@latest

2. Configure LLM (optional — works without config too)
   cachet config init

3. Capture a failure
   cachet capture --url POST:/pay --status 500 --error timeout

4. Diagnose it
   cachet ask --latest

5. Fix it, then verify
   cachet verify <failure-id>
──────────────────────────────────────────────────────────
```

## What to do when invoked

Read the user's args and load the relevant file from the `skill/` directory alongside this file:

| Invocation | Read |
|---|---|
| `/cachet setup` | `./skill/setup.md` |
| `/cachet setup dev` | `./skill/setup-dev.md` |
| `/cachet setup proxy` | `./skill/setup-proxy.md` |
| `/cachet setup watch` | `./skill/setup-watch.md` |
| `/cachet setup anthropic` | `./skill/setup.md` (anthropic section) |
| `/cachet setup openai` | `./skill/setup.md` (openai section) |
| `/cachet workflow` | `./skill/workflow.md` |
| `/cachet explain <cmd>` | `./skill/commands.md` |
| troubleshooting / errors | `./skill/troubleshooting.md` |
| no args | show Quick Start above, ask what they need |
