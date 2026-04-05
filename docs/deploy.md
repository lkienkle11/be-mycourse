# Deploying MyCourse (Backend + Frontend) on Ubuntu 24.04

This guide walks through **server setup**, **HTTPS for two hostnames on one machine**—the **apex / `www` domain for the Next.js frontend** (e.g. `yourdomain.net`) and **`api.` for the Go API** (e.g. `api.yourdomain.net`)—plus a **GitHub Actions–style CI/CD skeleton**. Replace **`yourdomain.net`** everywhere with your real **`.net`** domain. The workflow YAML is a **template**—add the real workflow file when you are ready.

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

Example layout:

```text
/opt/mycourse/be
/opt/mycourse/fe
```

1. Create the user/directory policy your org uses (dedicated `deploy` user recommended).
2. Clone or copy the repositories (or rely on CI to `rsync` artifacts later).
3. Ensure the backend path will contain `go.mod`, `config/`, and a `bin/` output directory.

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
- `API_KEY` if you use `/api/internal-v1`.

**`MIGRATE=1` behavior:** With the current `main.go`, enabling `MIGRATE=1` still runs `router.Run` after migrations (HTTP server starts). Typical approaches:

- Stop the running process (PM2/systemd), run once with `MIGRATE=1`, then return to normal startup; or
- Keep `MIGRATE=1` on every start if you accept running migrate on boot (safe when migrations are idempotent); or
- Later, add a dedicated `cmd/migrate` that only runs `Up` and exits—cleaner for CI.

---

### Step 11 — Build the backend binary

```bash
cd /opt/mycourse/be
go mod download
go build -o bin/mycourse-io-be -trimpath -ldflags="-s -w" .
```

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

### Step 16 — Run the API and web app under PM2

Example `ecosystem.config.cjs` on the server:

```javascript
module.exports = {
  apps: [
    {
      name: 'mycourse-api',
      cwd: '/opt/mycourse/be',
      script: './bin/mycourse-io-be',
      instances: 1,
      autorestart: true,
      max_memory_restart: '512M',
      env: { STAGE: 'prod' },
      env_file: '/opt/mycourse/be/.env',
    },
    {
      name: 'mycourse-web',
      cwd: '/opt/mycourse/fe',
      script: 'npm',
      args: 'run start',
      instances: 1,
      autorestart: true,
      env: { NODE_ENV: 'production', PORT: 3000 },
    },
  ],
};
```

```bash
pm2 start ecosystem.config.cjs
pm2 save
```

If your PM2 version does not support `env_file`, use **systemd** with `EnvironmentFile=` for secrets instead.

**Order note:** Start the Go API **after** `.env` points at reachable Postgres/Redis (**cloud or local**, Step 8), then rely on Nginx—Nginx only proxies to `127.0.0.1:8080`.

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
| 4 | `cache_clients.SetupRedis()` | Redis (auth/`/me` cache—see `docs/modules/auth.md`; **optional** if using hosted Redis or accepting DB-only fallback). |
| 5 | `MIGRATE=1` | `models.MigrateDatabase()` applies `migrations/*.up.sql`. |
| 6 | `config.InitSystem()`, `queues.Consume()` | Bootstrap and queue placeholder. |
| 7 | `api.InitRouter()` | Gin: CORS, gzip, `/api/v1` (JWT and public routes), `/api/internal-v1` (API key). |

**Smoke-test path:** `GET /api/v1/health` (no JWT). Auth routes live under `/api/v1/auth/...`; authenticated routes include `/api/v1/me` (see `api/v1/routes.go`).

**Config:** `config/app.yaml` + `config/app-<STAGE>.yaml`; `${VAR}` placeholders resolve from `.env` / environment in `pkg/setting`.

---

## Appendix C — CI/CD with GitHub Actions (after manual deploy works)

GitLab uses **GitLab CI** (`.gitlab-ci.yml`); the **same job split** applies—one concern per job, not one giant script.

### C.1 — Prepare the server for CI deploy (ordered)

1. Create a **deploy** SSH key pair; add the **public** key to `~/.ssh/authorized_keys` on the server.
2. Store the **private** key and connection details as GitHub **Secrets**: `SSH_PRIVATE_KEY`, `SSH_HOST`, `SSH_USER`, `DEPLOY_PATH`.
3. Ensure `DEPLOY_PATH` contains `be/bin/` and that PM2 can reload `mycourse-api`.

### C.2 — Pipeline principles

- **Lint/test/build** run on `ubuntu-latest`—the VPS does not need Go/Node for those steps if you ship binaries from CI.
- **Deploy** is its own job with `needs:` on successful build; it only **uploads artifacts** and **restarts** the process.
- **Migrate** stays a **separate** job (protected environment / manual approval). The current binary with `MIGRATE=1` still starts HTTP; maintain a server script such as `be/scripts/apply_migrations.sh` (stop PM2 → migrate → start) or add a dedicated migrate command later.

### C.3 — Example workflow (one job per task)

Place under `.github/workflows/deploy.yml` (adjust paths for a monorepo vs. `be`-only repo):

```yaml
name: CI/CD

on:
  push:
    branches: [main]

concurrency:
  group: deploy-${{ github.ref }}
  cancel-in-progress: true

jobs:
  prepare:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
      - name: Show git ref and short SHA
        run: |
          echo "REF=${{ github.ref_name }}"
          echo "SHA=${{ github.sha }}"
          git rev-parse --short HEAD

  lint-backend:
    runs-on: ubuntu-latest
    needs: [prepare]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: be/go.mod
          cache-dependency-path: be/go.sum
      - name: Go fmt check
        working-directory: be
        run: test -z "$(gofmt -l .)"

  test-backend:
    runs-on: ubuntu-latest
    needs: [lint-backend]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: be/go.mod
          cache-dependency-path: be/go.sum
      - name: Download modules
        working-directory: be
        run: go mod download
      - name: Run tests
        working-directory: be
        run: go test ./... -count=1 -short

  build-backend:
    runs-on: ubuntu-latest
    needs: [test-backend]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: be/go.mod
          cache-dependency-path: be/go.sum
      - name: Build Linux binary
        working-directory: be
        run: go build -trimpath -ldflags="-s -w" -o ../mycourse-io-be .
      - name: Upload binary artifact
        uses: actions/upload-artifact@v4
        with:
          name: mycourse-io-be
          path: mycourse-io-be

  build-frontend:
    runs-on: ubuntu-latest
    needs: [prepare]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: fe/package-lock.json
      - name: Install dependencies
        working-directory: fe
        run: npm ci
      - name: Build Next.js
        working-directory: fe
        run: npm run build
      - name: Upload frontend build artifact
        uses: actions/upload-artifact@v4
        with:
          name: fe-build
          path: fe/.next

  deploy-backend:
    runs-on: ubuntu-latest
    needs: [build-backend]
    environment: production
    steps:
      - name: Download binary
        uses: actions/download-artifact@v4
        with:
          name: mycourse-io-be
      - uses: webfactory/ssh-agent@v0.9.0
        with:
          ssh-private-key: ${{ secrets.SSH_PRIVATE_KEY }}
      - name: Add host to known_hosts
        run: ssh-keyscan -H "${{ secrets.SSH_HOST }}" >> ~/.ssh/known_hosts
      - name: Rsync binary to server
        run: |
          rsync -avz --chmod=755 ./mycourse-io-be \
            "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}:${{ secrets.DEPLOY_PATH }}/be/bin/mycourse-io-be"
      - name: Reload application on server
        run: |
          ssh "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}" \
            "pm2 reload mycourse-api || pm2 start ${{ secrets.DEPLOY_PATH }}/be/ecosystem.config.cjs"

  migrate-backend:
    runs-on: ubuntu-latest
    needs: [deploy-backend]
    environment: production-migrate
    steps:
      - uses: webfactory/ssh-agent@v0.9.0
        with:
          ssh-private-key: ${{ secrets.SSH_PRIVATE_KEY }}
      - name: Add host to known_hosts
        run: ssh-keyscan -H "${{ secrets.SSH_HOST }}" >> ~/.ssh/known_hosts
      - name: Invoke server-side migrate script
        run: |
          ssh "${{ secrets.SSH_USER }}@${{ secrets.SSH_HOST }}" \
            "bash ${{ secrets.DEPLOY_PATH }}/be/scripts/apply_migrations.sh"
```

| Job | Responsibility |
|-----|----------------|
| `prepare` | Checkout and record revision metadata. |
| `lint-backend` | Format/static checks (extend with `golangci-lint` later). |
| `test-backend` | `go test` before build/deploy. |
| `build-backend` | Produce Linux binary artifact. |
| `build-frontend` | Independent FE build artifact. |
| `deploy-backend` | Rsync binary + PM2 reload only. |
| `migrate-backend` | Controlled DB migration via your server script. |

If the repository root **is** the backend, drop the `be/` prefix in `working-directory` and artifact paths.

---

## Appendix D — Key files in repo `be`

| Area | Path |
|------|------|
| Entry | `main.go` |
| Router | `api/router.go`, `api/v1/routes.go` |
| Settings | `config/app.yaml`, `config/app-*.yaml`, `pkg/setting/setting.go` |
| DB / migrate | `models/setup.go`, `migrations/*.sql` |
| Cache | `cache_clients/redis.go`, `services/cache/` |
| Docs | `docs/modules/auth.md`, `docs/architecture.md` |

For HTTP/router relationships, GitNexus (repo `be`) queries such as *InitRouter Gin API* surface `InitRouter`, JWT middleware, and response helpers.

---

*Adjust paths, domains, and secrets to match your environment.*
