#!/usr/bin/env bash
# Deploy vocab-bot on a VPS with Docker Compose (external LLM).
# Run from vocab-bot directory. No secrets in this script — configure .env first.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$BOT_DIR"

if [[ ! -f .env ]]; then
  echo "Missing .env. Copy .env.example and set TELEGRAM_BOT_TOKEN, LLM_API_BASE, LLM_API_KEY, LLM_MODEL."
  echo "  cp .env.example .env && nano .env"
  exit 1
fi

if ! grep -q 'TELEGRAM_BOT_TOKEN=.\+' .env 2>/dev/null; then
  echo "Warn: TELEGRAM_BOT_TOKEN looks unset in .env"
fi
if ! grep -q 'LLM_API_BASE=.\+' .env 2>/dev/null; then
  echo "Warn: LLM_API_BASE looks unset in .env (use e.g. https://api.openai.com/v1 for production)"
fi

echo "Starting bot (external LLM from .env)..."
docker compose up -d --build
echo "Done. Logs: docker compose logs -f bot"
