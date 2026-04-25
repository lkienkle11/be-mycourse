# Deploying MyCourse (Backend + Frontend) on Ubuntu 24.04

This guide walks through **server setup**, **HTTPS for two hostnames on one machine**—the **apex / `www` domain for the Next.js frontend** (e.g. `yourdomain.net`) and **`api.` for the Go API** (e.g. `api.yourdomain.net`)—plus **CI/CD** (`.github/workflows/deploy-dev.yml` for the backend on **`master`**, and the frontend workflow in the **Next.js repo** on **`dev`** — checkout path on the server is often `/opt/mycourse/fe` or `fe-mycourse`). Replace **`yourdomain.net`** everywhere with your real **`.net`** domain.

The sections under **Deployment runbook** are ordered: follow **Step 1 → Step 2 → …** in sequence. Background context and CI/CD details come **after** the runbook.

---

## Deployment runbook (do these in order)

### Step 1 — Prerequisites

1. **Server:** A fresh or dedicated Ubuntu 24.04 LTS host with root/sudo access.
2. **DNS:** Point all public hostnames at this server’s IP **before** TLS:
   - **`yourdomain.net`** (apex) → **A** / **AAAA** (frontend).
   - **`www.yourdomain.net`** → **A** / **AAAA** (same IP; optional but recommended if you serve the site on `www`).
   - **`api.yourdomain.net`** → **A** / **AAAA** (same IP; backend API only).
3. **Data stores:** Decide per component:
   - **PostgreSQL** — usually **managed** (e.g. Supabase, RDS, Neon). The VPS then does **not** need the PostgreSQL *server* package; you only need connectivity + secrets in `.env`. Installing Postgres **on Ubuntu** is **optional** and only if you want the engine on the same machine.
   - **Redis** — **optional** for this stack when you use **hosted Redis** (or skip cache-heavy paths): point `REDIS_ADDR` at the cloud endpoint. A local `redis-server` on Ubuntu is **optional** and only if you want an on-box engine.
4. **API keys:** Fill `.env.example` / `.env.prod.example` (Supabase, JWT, etc.).

---

### Step 2 — Update the system and install core packages

The default `apt` list **does not** install the **PostgreSQL server** or **Redis server**—only `postgresql-client` for tooling. Add `postgresql` / `redis-server` in **Step 8** if you run those engines on this host; with **cloud-managed** DB/cache, skip those packages.

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y \
  ca-certificates curl wget git vim htop tmux unzip \
  build-essential pkg-config \
  ufw fail2ban \
  nginx certbot python3-certbot-nginx \
  software-properties-common apt-transport-https \
  postgresql-client jq rsync openssh-server
```

| Package | Purpose |
|---------|---------|
| `ca-certificates`, `curl`, `wget` | HTTPS, scripts, health checks |
| `git` | Clone repos; self-hosted runners (if any) |
| `vim`, `htop`, `tmux` | Operations |
| `build-essential`, `pkg-config` | CGO builds if you enable them later |
| `ufw` | Firewall |
| `fail2ban` | SSH brute-force mitigation (recommended) |
| `nginx` | Reverse proxy and TLS termination |
| `certbot`, `python3-certbot-nginx` | Let’s Encrypt certificates |
| `postgresql-client` | `psql` against **remote** or local Postgres (debug/migrate checks)—does **not** install the database server |
| `jq` | JSON in deploy/health scripts |
| `rsync` | Artifact sync from CI over SSH |
| `openssh-server` | Admin SSH and GitHub Actions deploy keys |

---

### Step 3 — Install Go (match `go.mod`)

The module targets **Go 1.25**. Prefer the official tarball over the distro package if versions differ.

```bash
GO_VER=1.25.0   # bump when go.mod changes
wget "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "go${GO_VER}.linux-amd64.tar.gz"
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh
go version
```

---

### Step 4 — Install Node.js (for the frontend and PM2)

TypeScript and Next.js are **not** separate system packages—install **Node LTS**, then install dependencies inside the `fe` repo.

**Option A — NodeSource (example: Node 22 LTS):**

```bash
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs
node -v && npm -v
```

**Option B — nvm:**

```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
# new shell
nvm install 22 && nvm use 22
```

---

### Step 5 — Install PM2 and enable startup

```bash
sudo npm install -g pm2
pm2 startup systemd -u "$USER" --hp "$HOME"
# Run once the `sudo env PATH=...` command PM2 prints
```

PM2 can run the **Go binary** or **`npm run start`** for Next.js.

---

### Step 6 — Install Docker (optional)

Only if you deploy with containers.

```bash
sudo apt install -y docker.io docker-compose-v2
sudo usermod -aG docker "$USER"
# Log out and back in for the `docker` group
```

---

### Step 7 — Configure the firewall

Do this **after** SSH access is stable so you are not locked out.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'   # HTTP + HTTPS
sudo ufw enable
sudo ufw status verbose
```

---

### Step 8 — PostgreSQL and Redis engines: managed cloud (typical) vs optional on this server

The app connects to Postgres and Redis **over the network**. If you already use **cloud-managed** databases (common with a rented VPS), you usually **do not** install the Postgres or Redis **daemon** on Ubuntu—only configure `.env` and network access (provider allowlists, TLS, etc.).

#### 8.1 — Managed Postgres + managed Redis (skip local engines)

**Do this path when** `DATABASE_URL` / `SUPABASE_DB_URL` and `REDIS_ADDR` point to **hosted** services.

1. **Do not** run `apt install postgresql` or `redis-server` on the app server unless you explicitly want them locally.
2. Keep **`postgresql-client`** from Step 2 so you can test from the VPS:

   ```bash
   psql "$DATABASE_URL" -c 'SELECT 1'
   ```

3. In the cloud provider’s console, allow outbound from this VPS (and **inbound** on the DB/Redis side if the product uses IP allowlists). Use the connection strings the vendor gives you in `.env`.

4. Set `REDIS_ADDR` to the **managed** host:port (and `REDIS_PASSWORD` if required). The codebase degrades to Postgres-only behaviour if Redis is unreachable, but you should still set a valid endpoint in production when you rely on cache.

#### 8.2 — Optional: install PostgreSQL server on this Ubuntu host

**Only if** you want Postgres **on the same machine** as the API (no cloud DB for the app DSN).

```bash
sudo apt install -y postgresql postgresql-contrib
sudo systemctl enable --now postgresql
```

Then create a role and database, set authentication in `/etc/postgresql/*/main/pg_hba.conf` (e.g. `scram-sha-256` for local TCP), and set `listen_addresses = 'localhost'` unless you need remote DB access. Build `DATABASE_URL` accordingly, e.g. `postgres://user:pass@127.0.0.1:5432/dbname`.

#### 8.3 — Optional: install Redis server on this Ubuntu host

**Skip** if `REDIS_ADDR` already targets **managed Redis**.

```bash
sudo apt install -y redis-server
sudo systemctl enable --now redis-server
redis-cli ping   # expect PONG
```

In `.env` for local Redis: `REDIS_ADDR=127.0.0.1:6379` (and `REDIS_PASSWORD` if you enable `requirepass` in `redis.conf`).

---

### Step 9 — Create deploy paths and place application code

**Actual server layout (as used in `ecosystem.config.cjs` and CI):**

```text
/var/www/be-mycourse/        ← backend root (cwd for all PM2 app entries)
  bin/
    mycourse-io-be-dev       ← dev binary (rsync'd by CI)
    mycourse-io-be-staging   ← staging binary
    mycourse-io-be-prod      ← production binary
  .env                       ← dev environment variables
  .env.staging               ← staging environment variables
  .env.prod                  ← production environment variables
  ecosystem.config.cjs       ← PM2 multi-environment config
  config/                    ← app.yaml + stage overrides
  migrations/                ← SQL migration files
```

```bash
# Create the directory structure on the server:
sudo mkdir -p /var/www/be-mycourse/bin
sudo chown -R "$USER":"$USER" /var/www/be-mycourse

# Clone (or rsync) the backend source — migrations, config, ecosystem file:
cd /var/www/be-mycourse
git clone https://github.com/your-org/mycourse.git .
# Or: git clone ... && cp -r mycourse/be/* /var/www/be-mycourse/
```

The `bin/` subdirectory is **created automatically** by the CI workflow via:
```bash
mkdir -p ${{ secrets.DEPLOY_PATH_DEV }}/bin
```

For the **frontend**, a separate path applies (see [`../fe-mycourse/docs/deploy.md`](../fe-mycourse/docs/deploy.md) when both repos live under the same parent directory). Frontend does not use `/var/www/be-mycourse`.

---

### Step 10 — Configure environment variables (backend)

On the server, add **non-committed** files per the README (e.g. `.env` + `.env.prod` for `STAGE=prod`).

Set at least:

- `STAGE=prod` (must match `config/app-prod.yaml` if you use it).
- `SERVER_PORT=8080`, `SERVER_RUN_MODE=release`.
- `DATABASE_URL` or full `DB_*` set—aligned with **cloud** or **local** Postgres (see Step 8).
- `SUPABASE_*`, `JWT_SECRET`, `APP_BASE_URL=https://api.yourdomain.net` (must match the public API URL).
- `CORS_ALLOWED_ORIGINS` — comma-separated **browser origins** for the frontend, e.g. `https://yourdomain.net,https://www.yourdomain.net` (no trailing slashes).
- `REDIS_ADDR` — **managed Redis** URL/host:port from your cloud provider, or `127.0.0.1:6379` only if you installed Redis on this host (Step 8.3). Omit or leave empty only if you accept cache falling back to DB-only behaviour.
- RBAC sync from constants is driven by **`/api/system`** (authenticated system JWT), not startup env flags. Populate **`system_app_config`** and **`system_privileged_users`** in Postgres as documented in `docs/architecture.md`.
- `CLI_REGISTER_NEW_SYSTEM_USER` — optional one-shot CLI to register the first privileged system user (process exits afterward).
- `API_KEY` if you use `/api/internal-v1`.

**`MIGRATE=1` behavior:** With the current `main.go`, enabling `MIGRATE=1` still runs `router.Run` after migrations (HTTP server starts). Typical approaches:

- Stop the running process (PM2/systemd), run once with `MIGRATE=1`, then return to normal startup; or
- Keep `MIGRATE=1` on every start if you accept running migrate on boot (safe when migrations are idempotent); or
- Later, add a dedicated `cmd/migrate` that only runs `Up` and exits—cleaner for CI.

---

### Step 11 — Build the backend binary

Binary names follow the environment convention used in `ecosystem.config.cjs`:

| Environment | Binary name | PM2 process name |
|-------------|-------------|-----------------|
| Dev (master) | `mycourse-io-be-dev` | `mycourse-api-dev` |
| Staging | `mycourse-io-be-staging` | `mycourse-api-staging` |
| Production | `mycourse-io-be-prod` | `mycourse-api-prod` |

**For a manual dev build on the server:**

```bash
cd /var/www/be-mycourse
go mod download
go build -trimpath -ldflags="-s -w" -o bin/mycourse-io-be-dev .
```

**CI builds** (see Appendix C) produce the binary in the GitHub Actions runner and `rsync` it directly to `DEPLOY_PATH_DEV/bin/`. You do **not** need Go installed on the server if you always deploy via CI.

> **Go version:** The project requires **Go 1.25.0** (match `go.mod` and the `go-version` in `.github/workflows/deploy-dev.yml`).

---

### Step 12 — Build the frontend (same host)

```bash
cd /opt/mycourse/fe
npm ci
npm run build
```

---

### Step 13 — Configure Nginx: frontend vs API on different upstreams (HTTP first)

Use **two separate site files** so each hostname proxies to a **different port** on the same machine:

| Hostname | Upstream | Role |
|----------|----------|------|
| `yourdomain.net`, `www.yourdomain.net` | `127.0.0.1:3000` | Next.js (PM2) |
| `api.yourdomain.net` | `127.0.0.1:8080` | Go API (PM2) |

1. Confirm DNS from Step 1 resolves to this host for **apex**, **`www`**, and **`api`**.
2. (Recommended) Disable the default site to avoid `server_name` clashes:

```bash
sudo rm -f /etc/nginx/sites-enabled/default
```

3. Create **`/etc/nginx/sites-available/mycourse-web`** — frontend, **port 80 only** for now:

```nginx
server {
    listen 80;
    server_name yourdomain.net www.yourdomain.net;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 90s;
    }
}
```

4. Create **`/etc/nginx/sites-available/mycourse-api`** — API, **port 80 only** for now:

```nginx
server {
    listen 80;
    server_name api.yourdomain.net;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 90s;
    }
}
```

5. Enable both sites and reload:

```bash
sudo ln -sf /etc/nginx/sites-available/mycourse-web /etc/nginx/sites-enabled/
sudo ln -sf /etc/nginx/sites-available/mycourse-api /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

---

### Step 14 — Obtain TLS certificates with Certbot (all hostnames in one certificate)

Request a **single certificate** that covers the frontend names and the API subdomain. Certbot’s nginx plugin will attach it to **both** vhosts.

```bash
sudo certbot --nginx \
  -d yourdomain.net \
  -d www.yourdomain.net \
  -d api.yourdomain.net
```

- Choose **redirect HTTP → HTTPS** when prompted so port 80 serves ACME/redirect and 443 serves the apps.
- Renewals are handled by `certbot.timer` on Ubuntu.

**Certificate path:** With multiple `-d` names, files usually live under `/etc/letsencrypt/live/<first-name>/` (here often `yourdomain.net`). Nginx snippets Certbot adds will point `ssl_certificate` and `ssl_certificate_key` at that directory for **both** site files. If you prefer a stable name regardless of order, add e.g. `--cert-name mycourse-net` to the command above (first run only).

**Reference — expected split after Certbot:**

- **`mycourse-web`:** `listen 443 ssl` for `server_name yourdomain.net www.yourdomain.net;` → `proxy_pass http://127.0.0.1:3000;` + same proxy headers as HTTP.
- **`mycourse-api`:** `listen 443 ssl` for `server_name api.yourdomain.net;` → `proxy_pass http://127.0.0.1:8080;` + same proxy headers as HTTP.

Both `server { ... 443 ... }` blocks should reference the **same** `ssl_certificate` and `ssl_certificate_key` paths (multi-SAN / multi-name cert).

**If you only use `www` for the site (no apex):** Omit `yourdomain.net` from `server_name` and from the `certbot` `-d` list; keep `api.yourdomain.net` as its own name on the cert.

---

### Step 16 — Run the API under PM2

The actual `ecosystem.config.cjs` (at `/var/www/be-mycourse/ecosystem.config.cjs`) manages **three environments** from one file using PM2's `env_file` feature:

```javascript
// /var/www/be-mycourse/ecosystem.config.cjs
module.exports = {
  apps: [
    {
      name: 'mycourse-api-dev',
      cwd: '/var/www/be-mycourse',
      script: './bin/mycourse-io-be-dev',
      instances: 1,
      autorestart: true,
      max_memory_restart: '1024M',
      env: {},
      env_file: '/var/www/be-mycourse/.env',
    },
    {
      name: 'mycourse-api-staging',
      cwd: '/var/www/be-mycourse',
      script: './bin/mycourse-io-be-staging',
      instances: 1,
      autorestart: true,
      max_memory_restart: '1024M',
      env: { STAGE: 'staging' },
      env_file: '/var/www/be-mycourse/.env.staging',
    },
    {
      name: 'mycourse-api-prod',
      cwd: '/var/www/be-mycourse',
      script: './bin/mycourse-io-be-prod',
      instances: 1,
      autorestart: true,
      max_memory_restart: '1024M',
      env: { STAGE: 'prod' },
      env_file: '/var/www/be-mycourse/.env.prod',
    },
  ],
};
```

**Key details:**

| Field | Value | Notes |
|-------|-------|-------|
| `cwd` | `/var/www/be-mycourse` | All three apps share the same working directory |
| `script` | `./bin/mycourse-io-be-{env}` | Binary in the `bin/` subdirectory |
| `env_file` | Absolute path to `.env`, `.env.staging`, `.env.prod` | Loaded by PM2 v5+ at startup |
| `max_memory_restart` | `1024M` | Process is restarted if it exceeds 1 GB RAM |
| `env.STAGE` | `staging` / `prod` | Inline env overrides merged on top of `env_file` |

**Starting and persisting:**

```bash
# Start only the dev process (typical after CI deploy):
pm2 start ecosystem.config.cjs --only mycourse-api-dev

# Or start all three environments:
pm2 start ecosystem.config.cjs

# Check status:
pm2 list
pm2 logs mycourse-api-dev --lines 50

# Persist across reboots:
pm2 save
```

**Reload (zero-downtime) after a binary update:**

```bash
pm2 reload mycourse-api-dev
# If the process does not exist yet, start it:
pm2 reload mycourse-api-dev || pm2 start ecosystem.config.cjs --only mycourse-api-dev
```

> **`env_file` requires PM2 v5+.** Run `pm2 --version` to verify. If you use an older PM2, inline the variables directly into the `env` block or use a systemd `EnvironmentFile=`.

**Order note:** Ensure `.env` (or the relevant stage file) points at reachable Postgres/Redis before starting the process. The API binds to `127.0.0.1:8080` (or as configured by `SERVER_HOST`/`SERVER_PORT`) — Nginx proxies `api.yourdomain.net` to that port.

---

### Step 17 — Verify end-to-end

```bash
curl -sS https://api.yourdomain.net/api/v1/health
```

Expect the JSON envelope described in the README (`code`, `message`, `status`).

Optional: open `https://yourdomain.net` (or `https://www.yourdomain.net`) in a browser and confirm the Next.js app loads through Nginx.

---

### Step 18 — Go-live checklist

- [ ] DNS for `yourdomain.net`, `www`, and `api` resolves to this host; Certbot succeeded for all names you use.
- [ ] `curl https://api.yourdomain.net/api/v1/health` returns HTTP 200.
- [ ] Frontend is reachable on `https://yourdomain.net` and/or `https://www.yourdomain.net` (as configured).
- [ ] `APP_BASE_URL` and outbound email (Brevo) use the correct public HTTPS URL.
- [ ] `CORS_ALLOWED_ORIGINS` matches the production web origin(s).
- [ ] Postgres (managed or local) accepts connections from this host; `psql` or the app can connect.
- [ ] Redis endpoint in `.env` is correct **or** you deliberately rely on no Redis (managed optional).
- [ ] Internal `API_KEY` is not committed; rotation process is defined.
- [ ] PM2 `startup` + `save` (or equivalent systemd units) are configured.
- [ ] `.env` backups and `JWT_SECRET` rotation if leaked.

---

## Appendix A — Target architecture

```text
Internet → DNS → Nginx (TLS, two server names) → {
  yourdomain.net, www.yourdomain.net  → 127.0.0.1:3000   (Next.js via PM2 or systemd)
  api.yourdomain.net                  → 127.0.0.1:8080   (Go binary or container)
}
```

- **Nginx:** One multi-name TLS certificate (Certbot) can serve **both** the frontend vhost and the API vhost; each `server_name` proxies to a **different** upstream port.
- **Go API:** Bind `127.0.0.1:8080` behind Nginx (recommended) or `0.0.0.0` if no local proxy—your choice.
- **Frontend:** `next build` + `next start` (or Docker) on `127.0.0.1:3000`.

---

## Appendix B — Backend startup and routing (codebase / GitNexus)

From the README and execution graph (GitNexus, repo **`be`**, query e.g. *HTTP router / startup*):

| Order | Component | Role |
|-------|-----------|------|
| 1 | `main.go` | `setting.Setup()` loads YAML + `.env` (with `STAGE`). |
| 2 | `models.Setup()` | PostgreSQL via `[database]` DSN. |
| 3 | Supabase packages | Separate Supabase DB URL + HTTP client when configured. |
| 4 | `pkg/cache_clients.SetupRedis()` | Redis (auth/`/me` cache—see `docs/modules/auth.md`; **optional** if using hosted Redis or accepting DB-only fallback). |
| 5 | `MIGRATE=1` | `models.MigrateDatabase()` applies `migrations/*.up.sql`. |
| 6 | `config.InitSystem()`, `queues.Consume()` | Bootstrap and queue placeholder. |
| 7 | `api.InitRouter()` | Gin: CORS, gzip, `/api/v1` (JWT and public routes), `/api/internal-v1` (API key). |

**Smoke-test path:** `GET /api/v1/health` (no JWT). Auth routes live under `/api/v1/auth/...`; authenticated routes include `/api/v1/me` (see `api/v1/routes.go`).

**Config:** `config/app.yaml` + `config/app-<STAGE>.yaml`; `${VAR}` placeholders resolve from `.env` / environment in `pkg/setting`.

---

## Appendix C — CI/CD with GitHub Actions

The active workflow is **`.github/workflows/deploy-dev.yml`**. It triggers on every push to **`master`**, builds the Go binary in CI (no Go installation needed on the server), and deploys via **`rsync`** of the binary into **`${DEPLOY_PATH_DEV}/bin/`**. Before replacing the binary, the server keeps a copy at **`bin/mycourse-io-be-dev.prev`**. After **`pm2 reload`**, CI runs **`scripts/pm2-reload-with-binary-rollback.sh`**, which polls **`GET /api/v1/health`** until the new process has finished startup (including everything in `main.go` before `router.Run`) and is listening. If the check times out, the script restores **`mycourse-io-be-dev.prev`** over the new binary, reloads PM2 again, prints **`pm2 logs`**, and exits with a failure so GitHub Actions shows a red deploy. **`git stash -u`**, **`git checkout master`**, and **`git pull`** run only after a successful health check, so a rolled-back server keeps the previous repo tree until a fixed binary deploys successfully.

### C.1 — Required GitHub Secrets

Set these in **Settings → Secrets and variables → Actions** on your GitHub repository:

| Secret | Description | Example |
|--------|-------------|---------|
| `SSH_PRIVATE_KEY` | Private key whose public half is in `~/.ssh/authorized_keys` on the server | PEM-format RSA/Ed25519 private key |
| `SSH_HOST` | Server IP address or hostname | `203.0.113.42` |
| `SSH_USER` | SSH login user | `ubuntu` / `deploy` |
| `DEPLOY_PATH_DEV` | Absolute path to the **backend** root on the server (same variable name as the frontend workflow; use a **different** path value per service) | `/var/www/be-mycourse` |

> **Setup:** Generate a deploy key pair (`ssh-keygen -t ed25519 -C "ci-deploy"`). Add the **public key** to `~/.ssh/authorized_keys` on the VPS. Add the **private key** as the `SSH_PRIVATE_KEY` secret.

### C.2 — Actual workflow: `deploy-dev.yml`

File: `.github/workflows/deploy-dev.yml`

**Trigger:** push to `master`.  
**Concurrency:** `cancel-in-progress: true` — a second push while the first is deploying cancels the in-flight run.  
**Job structure:** 2 sequential jobs — `build` → `deploy`.

```yaml
name: Deploy Backend to VPS (2 Jobs) Test deploy in Master Branch

on:
  push:
    branches:
      - master

concurrency:
  group: deploy-${{ github.ref }}
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Setup Go environment
        uses: actions/setup-go@v5
        with:
          go-version: "1.25.0"
          cache: true   # caches Go module downloads

      - name: Build Go Binary
        run: |
          go mod download
          go build -trimpath -ldflags="-s -w" -o mycourse-io-be-dev .

      - name: Upload Binary Artifact
        uses: actions/upload-artifact@v4
        with:
          name: backend-binary
          path: mycourse-io-be-dev
          retention-days: 1   # purged after 1 day to save GitHub storage

  deploy:
    runs-on: ubuntu-latest
    needs: build   # waits for 'build' to succeed before starting
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Download Binary Artifact
        uses: actions/download-artifact@v4
        with:
          name: backend-binary

      - name: Setup SSH Agent
        uses: webfactory/ssh-agent@v0.9.0
        with:
          ssh-private-key: ${{ secrets.SSH_PRIVATE_KEY }}

      - name: Add Server to known_hosts
        run: ssh-keyscan -H "${{ secrets.SSH_HOST }}" >> ~/.ssh/known_hosts

      - name: Ensure target directory exists
        run: |
          ssh "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}" \
            "mkdir -p ${{ secrets.DEPLOY_PATH_DEV }}/bin ${{ secrets.DEPLOY_PATH_DEV }}/scripts"

      - name: Copy deploy rollback helper to server
        run: |
          scp -q scripts/pm2-reload-with-binary-rollback.sh \
            "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}:${{ secrets.DEPLOY_PATH_DEV }}/scripts/pm2-reload-with-binary-rollback.sh"

      - name: Backup current binary on server (rollback target)
        run: |
          ssh "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}" \
            "cd ${{ secrets.DEPLOY_PATH_DEV }} && test -f bin/mycourse-io-be-dev && cp bin/mycourse-io-be-dev bin/mycourse-io-be-dev.prev || true"

      - name: Deploy Binary to Server via Rsync
        run: |
          rsync -avz --chmod=755 ./mycourse-io-be-dev \
            "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}:${{ secrets.DEPLOY_PATH_DEV }}/bin/mycourse-io-be-dev"

      - name: Reload PM2 with health gate and binary rollback
        run: |
          ssh "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}" \
            "chmod +x '${{ secrets.DEPLOY_PATH_DEV }}/scripts/pm2-reload-with-binary-rollback.sh' && \
             cd '${{ secrets.DEPLOY_PATH_DEV }}' && \
             DEPLOY_PATH='${{ secrets.DEPLOY_PATH_DEV }}' \
             PM2_APP_NAME=mycourse-api-dev \
             bash scripts/pm2-reload-with-binary-rollback.sh"
```

### C.3 — What each step does

| Step | Details |
|------|---------|
| **Checkout** | `build`: full clone for `go build`. `deploy`: clone so `scripts/pm2-reload-with-binary-rollback.sh` can be copied to the VPS. |
| **Setup Go** | Installs Go 1.25.0 with module cache enabled |
| **Build** | `go build -trimpath -ldflags="-s -w"` — stripped, reproducible binary named `mycourse-io-be-dev` |
| **Upload artifact** | Binary stored in GitHub's temporary artifact storage (1-day retention) |
| **Download artifact** | `deploy` job fetches the binary |
| **SSH agent** | Loads `SSH_PRIVATE_KEY` into the agent; no password prompt |
| **known_hosts** | `ssh-keyscan` prevents interactive host-key prompt |
| **mkdir -p** | Ensures `DEPLOY_PATH_DEV/bin/` and `DEPLOY_PATH_DEV/scripts/` exist |
| **scp rollback script** | Latest `scripts/pm2-reload-with-binary-rollback.sh` on the runner is pushed to the server (no need to `git pull` the script first). |
| **Backup binary** | If `bin/mycourse-io-be-dev` exists, copy to `bin/mycourse-io-be-dev.prev` for rollback. |
| **rsync** | Transfers `mycourse-io-be-dev` → `DEPLOY_PATH_DEV/bin/`, sets `chmod 755` |
| **PM2 + health + git** | Runs the rollback script: `pm2 reload` (or `pm2 start --only` if missing), polls `GET /api/v1/health` (default `http://127.0.0.1:8080/api/v1/health`, 90s). On success: `git stash -u && git checkout master && git pull`. On failure: restore `.prev` over the binary, reload PM2, print logs, exit non-zero. |

**Script environment (optional overrides on the server SSH line):**

| Variable | Default | Purpose |
|----------|---------|---------|
| `HEALTHCHECK_URL` | `http://127.0.0.1:8080/api/v1/health` | Must match bind address/port and API base path (`/api/v1` from Gin). |
| `ROLLBACK_HEALTH_TIMEOUT_SEC` | `90` | How long to wait for the process to survive startup and answer HTTP 200. |
| `BINARY_REL` | `bin/mycourse-io-be-dev` | Relative path to the binary under `DEPLOY_PATH`. |

### C.4 — Post-reload git pull (why?)

The final SSH command also syncs the repository on the server:

```bash
git stash -u   # stash any local changes (including untracked files)
git checkout master
git pull
```

This keeps `config/`, `migrations/`, and `ecosystem.config.cjs` on the server in sync with the `master` branch — so new YAML config files or SQL migrations are available without a separate manual pull.

### C.5 — Pipeline principles and future extensions

- **Build in CI, not on server:** The VPS does not need Go installed for normal deploys. The binary is built reproducibly in the GitHub Actions runner.
- **`cancel-in-progress: true`:** Rapid successive pushes to `master` only deploy the latest commit, avoiding partial deploys.
- **Migrations:** Currently not automated in CI. Recommended approach: stop PM2, run binary once with `MIGRATE=1`, restart. Add a separate `migrate` workflow with `environment: production-migrate` (requires manual approval) when you need controlled migrations.
- **Staging / Production:** To extend CI for staging/production environments, add jobs that build `mycourse-io-be-staging` / `mycourse-io-be-prod` and reload `mycourse-api-staging` / `mycourse-api-prod` respectively — following the same `rsync` + `pm2 reload --only <name>` pattern.
- **Frontend CI:** Separate repo/workflow — push to **`dev`**, secret **`DEPLOY_PATH_DEV`** (FE checkout path), server **`npm ci` + `npm run build`** + PM2 **`mycourse-web-dev`** — see [`../fe-mycourse/docs/deploy.md`](../fe-mycourse/docs/deploy.md) Appendix G.

---

## Appendix D — Key files in repo `be`

| Area | Path | Notes |
|------|------|-------|
| Entry point | `main.go` | `setting.Setup()` → DB → Redis → router → listen |
| Router | `api/router.go`, `api/v1/routes.go` | Gin: CORS, gzip, JWT middleware, public + private routes |
| Settings | `config/app.yaml`, `config/app-*.yaml`, `pkg/setting/setting.go` | YAML + `.env` merge via `STAGE` env var |
| DB / migrate | `models/setup.go`, `migrations/*.sql` | Run with `MIGRATE=1`; applied on startup |
| Cache | `pkg/cache_clients/redis.go`, `services/cache/` | Redis client; degrades gracefully if unavailable |
| Error codes | `pkg/errcode/codes.go`, `pkg/errcode/messages.go` | Mirrors `src/types/api.ts` (ApiErrorCode) in the FE |
| HTTP errors | `pkg/httperr/middleware.go` | Global Gin error handler |
| CI/CD | `.github/workflows/deploy-dev.yml` | Active 2-job workflow (build → deploy on master) |
| Deploy rollback | `scripts/pm2-reload-with-binary-rollback.sh` | Health-gated `pm2 reload`; restores `bin/mycourse-io-be-dev.prev` on failure |
| PM2 config | `ecosystem.config.cjs` | 3-environment PM2 config (`dev`, `staging`, `prod`) |
| Docs | `docs/modules/auth.md`, `docs/architecture.md` | Module-level docs |

**Server paths (matching `ecosystem.config.cjs`):**

| File/Dir | Server path |
|----------|------------|
| Backend root (`cwd`) | `/var/www/be-mycourse/` |
| Dev binary | `/var/www/be-mycourse/bin/mycourse-io-be-dev` |
| Staging binary | `/var/www/be-mycourse/bin/mycourse-io-be-staging` |
| Production binary | `/var/www/be-mycourse/bin/mycourse-io-be-prod` |
| Dev env file | `/var/www/be-mycourse/.env` |
| Staging env file | `/var/www/be-mycourse/.env.staging` |
| Production env file | `/var/www/be-mycourse/.env.prod` |

For HTTP/router relationships, GitNexus (repo `be`) queries such as *InitRouter Gin API* surface `InitRouter`, JWT middleware, and response helpers.

---

*Adjust paths, domains, and secrets to match your environment.*
