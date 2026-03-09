# vocab-bot

Telegram bot for collocation training. Uses SQLite and an LLM for generating collocations and grading answers. All UI and task instructions are in English.

## Training flow

1. **You send a list of words** (e.g. via `/add` → "deadline, meeting").
2. **The LLM generates collocations** for those words; they are stored with level 1.
3. **Each collocation has its own level** (1 → 2 → 3 → mastered). When you train, the bot picks a collocation and shows the exercise for *that* collocation’s current level.
4. **Exercise type = level:** the same exercise type is used for all collocations at that level (see table below). After a correct answer, that collocation’s level increases; after wrong, it stays and is scheduled sooner.
5. **Grading** uses the level-based task (see `GradePrompt` in `internal/llm/prompts.go`). A native example is shown after every answer (including correct ones).

## Exercise levels

See [docs/EXERCISE_LEVELS_SPEC.md](docs/EXERCISE_LEVELS_SPEC.md) for the full spec. Summary:

| Level | Exercise type | Task |
|-------|----------------|------|
| 1 | Meaning | Explain what the collocation means (in English). |
| 2 | Gap | Complete the sentence with the collocation (in English). |
| 3 | Fill | Use the collocation in one sentence (in English). |
| 4 | Paraphrase | Rewrite a given sentence using the collocation (in English). |
| mastered | Refresh | Gap-fill for retention. |

Only one exercise type per level; all collocations at level 1 get the level-1 task, etc.

## Quick start (local)

```bash
cp .env.example .env   # set TELEGRAM_BOT_TOKEN, LLM_API_BASE, LLM_API_KEY
go run ./cmd/bot
```

## Docker Compose

1. **Configure env** (once):
   ```bash
   cp .env.example .env
   # Edit .env: TELEGRAM_BOT_TOKEN, LLM_API_BASE, LLM_MODEL, LLM_API_KEY
   ```
   For a local LLM see [docs/LLM.md](docs/LLM.md) (e.g. Ollama).

2. **Run the bot:**
   ```bash
   docker compose up -d --build
   docker compose logs -f bot
   ```
   Use `--build` so the image is rebuilt after code changes. SQLite data is in the `bot-data` volume.

3. **Bot + MCP HTTP API** (for scripts / Cursor):
   ```bash
   docker compose --profile mcp up -d
   ```
   MCP server runs on port 8080, same DB volume.

4. **Stop:** `docker compose down`. Data persists. `docker compose down -v` removes the volume.

## Production deployment (VPS + external LLM)

On a VPS the bot uses an **external LLM** (OpenAI or compatible API); no LLM runs on the server. In `.env` set `LLM_API_BASE=https://api.openai.com/v1`, `LLM_API_KEY=sk-...`, `LLM_MODEL=gpt-4o-mini`. See [docs/LLM.md](docs/LLM.md).

See **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)** for:

- Creating a Telegram bot (BotFather) and getting the token
- Choosing a VPS (Hetzner, DigitalOcean, Vultr, etc.) and preparing the server
- Configuring env and running with Docker Compose or K8s
- [docs/SECURITY.md](docs/SECURITY.md) — secrets, input limits, firewall

## MCP HTTP API (optional)

With Docker: `docker compose --profile mcp up -d`. Without Docker:

```bash
MCP_PORT=8080 go run ./cmd/mcp
```

Endpoints: `POST /add_words`, `POST /next_exercise`, `POST /grade_answer`, `GET /stats?chat_id=...`, `GET /health`.

## Database and cleanup

- One SQLite file; no separate DB server. **Several users** can use the same bot: vocabulary and stats are shared (one pool). For per-user decks you’d need a schema change (see `docs/DEPLOYMENT.md`).
- **`/cleanup`** — deletes attempt records older than 90 days to keep the DB small; progress and collocations are unchanged.

## Project layout

- `cmd/bot` – Telegram bot entrypoint  
- `cmd/mcp` – MCP HTTP API entrypoint (optional)  
- `internal/` – config, db, domain, srs, llm, trainer, bot, mcp  
- `k8s/` – Kubernetes manifests  
- `docs/DEPLOYMENT.md` – deployment instructions  
