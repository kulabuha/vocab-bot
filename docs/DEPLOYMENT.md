# Deploying vocab-bot to production

You need: **a server** (VPS or cloud VM) where the bot runs, and a **Telegram bot** (token from BotFather). On the VPS you use an **external LLM** (OpenAI or compatible API); the bot does not run an LLM on the server.

---

## Quick: VPS + external LLM

1. **Get credentials:** Telegram token from @BotFather; OpenAI (or other) API key.
2. **Provision a VPS** (e.g. Ubuntu 22.04, 1 vCPU, 512 MB–1 GB RAM).
3. **On the server:**

```bash
# Install Docker + Compose
curl -fsSL https://get.docker.com | sh
apt-get update && apt-get install -y docker-compose-plugin

# Deploy bot
cd /opt && git clone https://github.com/YOUR_USER/my-collocation-learn-buddy.git
cd my-collocation-learn-buddy/vocab-bot
cp .env.example .env
nano .env   # set TELEGRAM_BOT_TOKEN, LLM_API_BASE, LLM_API_KEY, LLM_MODEL
docker compose up -d --build
docker compose logs -f bot
```

4. **In `.env`** set at least:
   - `TELEGRAM_BOT_TOKEN` — from BotFather
   - `LLM_API_BASE=https://api.openai.com/v1` (or your provider’s base URL)
   - `LLM_API_KEY=sk-...` (your API key)
   - `LLM_MODEL=gpt-4o-mini` (or e.g. `gpt-4o`)

No LLM runs on the VPS; all LLM calls go over HTTPS to the external API. See [docs/LLM.md](LLM.md) for other providers (Azure, Groq, etc.).

Optional: from `vocab-bot/` run `./scripts/deploy-vps.sh` to start the bot (checks `.env` exists, then runs `docker compose up -d --build`).

---

## How to deliver code to the VPS (recommended: Git)

**Recommended: use GitHub (or GitLab, etc.) and deploy from the repo.**

| Method | Pros | Cons |
|--------|------|------|
| **Git clone / pull** | Version control, easy updates (`git pull` + rebuild), canonical for teams | Requires repo to be pushed first |
| Copy archive (zip/tar) | No Git on server needed | No easy updates, no history, manual copy each time |

**Canonical flow:**

1. **Store code on GitHub** (create repo, push your project). Keep `.env` out of the repo (use `.env.example` only).
2. **First deploy on VPS:** clone, configure, run:
   ```bash
   git clone https://github.com/YOUR_USER/my-collocation-learn-buddy.git
   cd my-collocation-learn-buddy/vocab-bot
   cp .env.example .env && nano .env   # set secrets
   docker compose up -d --build
   ```
3. **Updates:** on the VPS, pull and rebuild:
   ```bash
   cd /path/to/my-collocation-learn-buddy
   git pull
   cd vocab-bot && docker compose up -d --build
   ```
   Or use `./scripts/deploy-vps.sh` after `git pull` if you're already in `vocab-bot/`.

So: **push code to GitHub, then on the VPS always deploy from the repo** (clone once, then pull + rebuild for updates). Avoid copying archives for ongoing deploys.

**First-time push to GitHub:** the repo has a root `.gitignore` (ignores `.env`, `*.db`, binaries, `.DS_Store`, etc.). From the project root:

```bash
git init
git add .
git status   # confirm .env and *.db are not staged
git commit -m "Initial commit: vocab-bot"
git branch -M main
git remote add origin https://github.com/YOUR_USER/my-collocation-learn-buddy.git
git push -u origin main
```

Ensure `.env` is never committed; only `.env.example` is tracked.

---

## Automated deploy with GitHub Actions

You can deploy automatically on every push to `main` using the included workflow.

### 1. One-time setup on the VPS

- Clone the repo (e.g. to `/opt/my-collocation-learn-buddy`) and do the **first deploy by hand**: create `.env`, run `docker compose up -d --build` once in `vocab-bot/`.
- Ensure the VPS can **pull** from GitHub:
  - **Private repo:** add the VPS SSH public key as a **Deploy key** in the repo (Settings → Deploy keys), or use a **Personal Access Token** and `git config credential.helper` / HTTPS URL with token.
  - **Public repo:** no extra step; `git pull` will work.
- Create an **SSH key pair** for Actions (do not use your personal key). On your machine:
  ```bash
  ssh-keygen -t ed25519 -C "github-actions-deploy" -f deploy_key -N ""
  ```
  Add **`deploy_key.pub`** to the VPS: `cat deploy_key.pub >> ~/.ssh/authorized_keys` (for the user that will run the deploy). Keep **`deploy_key`** (private) for the next step.

### 2. GitHub repo secrets

In the repo: **Settings → Secrets and variables → Actions → New repository secret.** Add:

| Secret            | Description |
|-------------------|-------------|
| `VPS_HOST`        | VPS IP or hostname (e.g. `123.45.67.89` or `vps.example.com`) |
| `SSH_USER`        | SSH user (e.g. `root` or `deploy`) |
| `SSH_PRIVATE_KEY` | Full content of the **private** key file (e.g. `deploy_key`) |
| `DEPLOY_PATH`     | *(Optional)* Path to the repo on the VPS. Default: `/opt/my-collocation-learn-buddy` |

### 3. What the pipeline does

On **push to `main`** (or when you run the workflow manually from the Actions tab):

1. The workflow SSHs into the VPS.
2. Runs `cd $DEPLOY_PATH && git pull && cd vocab-bot && docker compose up -d --build`.
3. The bot restarts with the new code; SQLite data in the `bot-data` volume is kept.

No secrets or `.env` are in the repo; the VPS already has `.env` from the first manual deploy.

### 4. Manual run

In the repo: **Actions → Deploy to VPS → Run workflow.** Use this to redeploy without pushing.

---

## 1. Create a Telegram bot and get the token

1. Open Telegram and search for **@BotFather**.
2. Send: `/newbot`
3. Follow the prompts:
   - **Name** – e.g. "My Collocation Buddy" (display name).
   - **Username** – must end in `bot`, e.g. `my_collocation_buddy_bot`.
4. BotFather replies with a **token** like `123456789:ABCdefGHI...`. **Copy and store it securely** — this is your `TELEGRAM_BOT_TOKEN`.
5. (Optional) Set description: `/setdescription` and choose your bot, then send a short description for users.

You’ll use this token on the server as the `TELEGRAM_BOT_TOKEN` environment variable.

---

## 2. Choose a server

You need a machine that:

- Is reachable from the internet (for Telegram API).
- Can run Docker (recommended) or a Go binary.
- Has a small amount of disk (SQLite + binary; 1 GB is enough to start).

**Common options:**

| Option | Typical use | Notes |
|--------|-------------|--------|
| **VPS** (DigitalOcean, Hetzner, Linode, Vultr, etc.) | Simple, cheap | Small droplet/instance (1 vCPU, 512 MB–1 GB RAM). Install Docker, run with Docker Compose. |
| **Cloud VM** (AWS EC2, GCP Compute, Azure VM) | If you already use the cloud | Same idea: install Docker, run the bot. |
| **Kubernetes** (GKE, EKS, or own cluster) | If you already have K8s | Use the manifests in `k8s/`. |
| **Home / office server** | If you have a always-on machine | Ensure it has a stable IP or use a tunnel (e.g. ngrok for testing only). |

For a first production deployment, a **small VPS + Docker Compose** is the simplest.

### Where to get a VPS

| Provider | Notes | Typical price |
|----------|--------|----------------|
| **Hetzner** | Good value in EU/US; simple panel | ~€4–5/mo for CX11/CX21 |
| **DigitalOcean** | Simple, good docs | ~$6/mo for basic droplet |
| **Vultr** | Many regions | ~$6/mo |
| **Linode (Akamai)** | Reliable | ~$5/mo |
| **Oracle Cloud** | Free tier (ARM) | Free (with limits) |

**Recommendation:** For EU/RU latency, **Hetzner** or **DigitalOcean** (Frankfurt/AMS). For US, any of the above. Create a small instance (1 vCPU, 512 MB–1 GB RAM is enough; the bot is light and only calls external APIs).

---

## 3. Prepare the server (VPS + Docker Compose)

### 3.1 Create a VPS and SSH in

- Create a droplet/instance (e.g. Ubuntu 22.04).
- SSH in: `ssh root@YOUR_SERVER_IP` (or use a non-root user with sudo).

### 3.2 Install Docker and Docker Compose

On Ubuntu/Debian:

```bash
# Docker
curl -fsSL https://get.docker.com | sh
# Docker Compose (plugin)
apt-get update && apt-get install -y docker-compose-plugin
# Verify
docker --version
docker compose version
```

### 3.3 Deploy the bot

**Option A: Clone the repo on the server**

```bash
cd /opt
git clone https://github.com/YOUR_USER/my-collocation-learn-buddy.git
cd my-collocation-learn-buddy/vocab-bot
```

**Option B: Build on your machine and push image**

On your laptop (from the repo root where `vocab-bot/` lives):

```bash
cd vocab-bot
docker build -t your-registry/vocab-bot:latest .
docker push your-registry/vocab-bot:latest
```

On the server, create `docker-compose.yml` (or copy it) and set `image: your-registry/vocab-bot:latest` instead of `build: .`.

### 3.4 Configure environment (external LLM on VPS)

On the server, in the same directory as `docker-compose.yml`:

```bash
cp .env.example .env
nano .env   # or vim
```

Set at least:

- `TELEGRAM_BOT_TOKEN` – token from BotFather.
- **External LLM** (recommended on VPS; no LLM runs on the server):
  - `LLM_API_BASE` – e.g. `https://api.openai.com/v1` for OpenAI.
  - `LLM_API_KEY` – API key (e.g. from platform.openai.com or your provider).
  - `LLM_MODEL` – e.g. `gpt-4o-mini` or `gpt-4o` for OpenAI.

For other providers (Azure OpenAI, Groq, Together, etc.) use their base URL and model name; the bot uses the same OpenAI-compatible `/chat/completions` API. See [LLM.md](LLM.md).

Save and exit. **Do not commit `.env` to git.** See `docs/SECURITY.md` for more.

### 3.5 Run the bot

```bash
docker compose up -d
docker compose ps
docker compose logs -f bot
```

If the bot is running, you’ll see logs (e.g. “started” or “polling”). In Telegram, open your bot and send `/start` or `/add` to test.

### 3.6 Persistence and restarts

- Data is in the `bot-data` Docker volume (SQLite under `/data` in the container).
- On reboot, run again: `docker compose up -d` (or enable a systemd unit that runs `docker compose up -d` in the project dir).

---

## 4. Check that it works in production

1. **Logs**  
   `docker compose logs -f bot` — no repeated errors; you may see “update received” or similar when you write to the bot.

2. **Telegram**  
   - Open your bot by username (e.g. `@my_collocation_buddy_bot`).
   - Send `/start` or `/add`. You should get a reply (even if it’s “send me words” or “not implemented yet” if that part is still stubbed).

3. **Process**  
   `docker compose ps` — container state should be `Up`.

4. **Data**  
   After using `/add` and `/train`, the DB grows. Use **`/cleanup`** in the bot to delete old attempt records (older than 90 days); progress and collocations are unchanged. This keeps the SQLite file from growing without bound.

---

## 4.1 Database and multi-user

- **SQLite** is the only store; one file (e.g. `/data/app.db` in Docker). No extra DB server.
- **Current design: one shared vocabulary.** Collocations and stats are global (not per user). Several users can use the same bot; they share the same collocation pool and see the same stats. This is enough for a single learner or a small shared bot.
- **Per-user decks** would require a schema change (e.g. `chat_id` on `collocations` and filtering all queries). Not in scope for the current MVP.
- **Cleanup:** `/cleanup` deletes attempt history older than 90 days to limit DB size. Collocations and progress (level, next_due) are not removed.

---

## 5. Optional: run without Docker (binary on server)

If you prefer not to use Docker:

1. On your machine: `cd vocab-bot && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o vocab-bot ./cmd/bot`
2. Copy `vocab-bot` and (if needed) migrations to the server, e.g. `/opt/vocab-bot/`.
3. Set env (e.g. in a systemd unit or shell):  
   `DB_PATH=/opt/vocab-bot/data/app.db`, `TELEGRAM_BOT_TOKEN=...`, `LLM_API_BASE=...`, `LLM_API_KEY=...`
4. Run: `./vocab-bot`
5. For production, use systemd so it restarts on failure and on reboot.

---

## 6. Kubernetes (if you use K8s)

1. Build and push the image:  
   `docker build -t your-registry/vocab-bot:latest .` and `docker push ...`
2. Create the secret (do **not** commit real values):  
   `kubectl create secret generic vocab-bot-secrets --from-literal=TELEGRAM_BOT_TOKEN=xxx --from-literal=LLM_API_KEY=xxx`
3. Edit `k8s/deployment.yaml` and set `image: your-registry/vocab-bot:latest`.
4. Apply:  
   `kubectl apply -f k8s/`
5. Check: `kubectl get pods`, `kubectl logs -f deployment/vocab-bot`

---

## 7. Security checklist

- **Never** commit `.env` or real tokens to git.
- Prefer creating the K8s secret with `kubectl create secret` (or a secret manager) instead of committing `k8s/secret.yaml` with real data.
- Restrict firewall: only SSH (and any admin ports you need); the bot only needs outbound HTTPS to Telegram and the LLM API.
- Keep the OS and Docker/K8s up to date.

See **`docs/SECURITY.md`** for input limits, secrets handling, and network guidance.

---

## Quick reference

| Step | Action |
|------|--------|
| 1 | Get token from @BotFather → `TELEGRAM_BOT_TOKEN` |
| 2 | Get LLM API key (e.g. OpenAI) → `LLM_API_KEY`, set `LLM_API_BASE` |
| 3 | Provision a server (VPS or cloud VM) |
| 4 | Install Docker + Docker Compose |
| 5 | Clone/copy `vocab-bot` and set `.env` |
| 6 | Run `docker compose up -d` and test in Telegram |

Once the bot binary and handlers are fully implemented, the same flow applies: set token and LLM keys, run with Docker or K8s, and verify with `/add` and `/train` in Telegram.
