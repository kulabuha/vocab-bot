# Security

## Secrets

- **Never** commit `.env` or real tokens/API keys to git. Use `.env.example` as a template only.
- `TELEGRAM_BOT_TOKEN` and `LLM_API_KEY` are loaded from environment only; they are not logged or printed.
- On a VPS or production: set env via your process manager (systemd, Docker env file, K8s secrets). Do not pass secrets on the command line in shared shells.

## Input limits

- **Adding words** (`/add`): max 2000 characters, max 50 words per message (excess dropped). Reduces abuse and keeps LLM requests bounded.
- **Training answers**: max 2000 characters per message. Prevents oversized payloads to the grading API.

## Network

- The bot only needs **outbound** HTTPS to Telegram API and to the LLM API. No inbound ports are required for the Telegram bot (long polling).
- If you run the MCP HTTP server, expose it only to trusted networks or put it behind a reverse proxy with auth/rate limiting.
- Restrict firewall: allow SSH; allow outbound 443; do not open unnecessary ports.

## Deployment

- Run the app as a non-root user when possible (e.g. in Docker with a USER directive, or on the host with a dedicated user).
- Keep the OS, Docker, and Go binary dependencies updated.
- Prefer a minimal base image for Docker to reduce attack surface.
