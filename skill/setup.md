# /cachet setup — LLM provider configuration

Guide the user through creating `cachet.config.json` in their project root.

## Step 1 — Check if config exists

```bash
ls cachet.config.json 2>/dev/null && echo "EXISTS" || echo "MISSING"
```

If it exists, read it and show current settings. Ask if they want to update it.

## Step 2 — Choose a provider

Ask: "Which LLM provider do you want to use?"
- **anthropic** — Claude (Sonnet 4.6 recommended)
- **openai** — GPT-4o or any OpenAI-compatible endpoint
- **none** — no-config mode: cachet prints the prompt to stdout so you can paste it anywhere

## Step 3 — Write the config

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

**None (stdout mode):**
```json
{ "provider": "none" }
```

Write the config with the user's actual key. Remind them to gitignore it:

```bash
echo "cachet.config.json" >> .gitignore
```

## Step 4 — Environment variable alternative

If the user doesn't want to store the key in a file:

```bash
export CACHET_PROVIDER=anthropic
export CACHET_API_KEY=sk-ant-...
export CACHET_MODEL=claude-sonnet-4-6
```

Env vars take precedence over `cachet.config.json`.
