# PharmaSense

> **Save €1000s monthly by catching at-risk pharmacy products before they expire.**

PharmaSense is a production-quality multi-tenant SaaS for pharmacies in Cyprus. It monitors product expiry dates, calculates risk levels, and sends email/SMS alerts — helping pharmacies reduce waste and avoid financial loss.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.23 · Gin · PostgreSQL 18 · golang-migrate · JWT |
| Frontend | Next.js 15 (App Router) · TypeScript · Tailwind CSS v3 · next-intl |
| Email | Resend (mock mode when key is empty) |
| SMS | Twilio (mock mode when key is empty) |
| Payments | Stripe (mock mode when key is empty) |
| Local DB | Docker (Postgres only) |
| Deploy | Railway (Railpack) |

---

## Local Development Setup

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker Desktop
- (Optional) [migrate CLI](https://github.com/golang-migrate/migrate) for manual migrations

### 1. Clone and configure

```bash
git clone https://github.com/your-org/pharmasense.git
cd pharmasense
cp backend/.env.example backend/.env
```

The `.env` defaults work for local dev with no external accounts needed.

### 2. Start PostgreSQL (Docker)

```bash
make db-up
```

This starts a single Postgres 18 container on `localhost:5432`. The backend and frontend run **natively** (not in Docker).

### 3. Install dependencies and seed

```bash
make setup         # go mod download + npm install
make seed          # runs migrations + seeds demo data
```

`make seed` is idempotent — safe to re-run.

### 4. Start services (two terminals)

**Terminal 1 — Backend:**
```bash
make dev-backend   # Gin on :3001
```

**Terminal 2 — Frontend:**
```bash
make dev-frontend  # Next.js on :3000
```

Open http://localhost:3000 and log in with:
| Email | Password | Role |
|---|---|---|
| `admin@pharmasense.cy` | `Demo1234!` | Admin (Nicosia Central) |
| `chain_admin@pharmasense.cy` | `Demo1234!` | Chain Admin (all 3 pharmacies) |
| `staff@pharmasense.cy` | `Demo1234!` | Staff (Nicosia Central) |

### Health check

```bash
curl localhost:3001/api/v1/healthz
# {"status":"ok"}
```

---

## Makefile Reference

| Command | Description |
|---|---|
| `make setup` | Download all deps (Go + Node) |
| `make db-up` | Start Postgres Docker container |
| `make db-down` | Stop Postgres container |
| `make seed` | Wipe + re-seed demo data |
| `make dev-backend` | Run Gin server on :3001 |
| `make dev-frontend` | Run Next.js on :3000 |
| `make test` | Run Go tests |
| `make build` | Build production binaries |
| `make migrate-up` | Apply migrations manually |
| `make migrate-down` | Roll back one migration |

---

## Connecting pgAdmin 4 to Local Postgres

pgAdmin 4 is a native desktop app (NOT in Docker).

**Install:**
- macOS: `brew install --cask pgadmin4`
- Windows: bundled with the PostgreSQL installer from pgadmin.org
- Linux: see https://www.pgadmin.org/download/

**Connect:**
1. Open pgAdmin 4 → set master password on first launch.
2. Right-click **Servers** → **Register** → **Server**.
3. **General** tab → Name: `PharmaSense Local`.
4. **Connection** tab:
   - Host: `localhost`
   - Port: `5432`
   - Maintenance database: `pharmasense`
   - Username: `postgres`
   - Password: `postgres` (check "Save Password")
5. **Save** → expand server → Databases → pharmasense → Schemas → public → Tables.

---

## Connecting pgAdmin 4 to Railway Postgres (Production)

1. Railway dashboard → Postgres service → **Variables** tab.
2. Copy `DATABASE_PUBLIC_URL`. Format: `postgresql://user:password@host:port/database`.
3. pgAdmin → Register Server → fill fields from the URL.
4. **SSL tab** → SSL mode: `require`.
5. Save and inspect production data.

---

## Deploy to Railway

### 1. Push to GitHub

Create a repo and push the code.

### 2. Create a Railway project

https://railway.com → New Project → Empty Project → name it `pharmasense`.

### 3. Add PostgreSQL 18

- \+ Create → Database → PostgreSQL.
- Ensure it's version 18. If the default is older, use: https://railway.com/deploy/postgres-18-ssl
- Railway provisions it and injects `DATABASE_URL`.

### 4. Add Backend service

- \+ Create → GitHub Repo → select your repo.
- Settings → Root Directory: `/backend`
- Settings → Watch Paths: `/backend/**`
- Variables → add all backend vars from the table below. Use `${{ Postgres.DATABASE_URL }}` for the DB.
- Networking → Generate Domain.

### 5. Add Frontend service

- \+ Create → GitHub Repo → same repo.
- Settings → Root Directory: `/frontend`
- Settings → Watch Paths: `/frontend/**`
- Variables: `NEXT_PUBLIC_API_URL = https://${{ Backend.RAILWAY_PUBLIC_DOMAIN }}`
- Networking → Generate Domain.

### 6. Configure Stripe webhook

In the Stripe dashboard, add a webhook endpoint:
```
https://<backend-domain>/api/v1/billing/webhook
```
Copy the signing secret → set `STRIPE_WEBHOOK_SECRET` on the backend service.

### 7. Wait for green deploys

The backend auto-migrates on startup. No manual migration step needed.

### 8. Seed demo data (optional, one-time)

```bash
npm install -g @railway/cli
railway login
railway link
railway run --service backend go run ./cmd/seed
```

### 9. Open the frontend domain

Log in with `admin@pharmasense.cy` / `Demo1234!` or sign up a new account.

---

## Railway Environment Variables

### Backend service

| Variable | Value |
|---|---|
| `DATABASE_URL` | `${{ Postgres.DATABASE_URL }}` |
| `JWT_SECRET` | `openssl rand -base64 32` output |
| `JWT_EXPIRY_HOURS` | `24` |
| `FRONTEND_ORIGIN` | `https://${{ Frontend.RAILWAY_PUBLIC_DOMAIN }}` |
| `ENV` | `production` |
| `RESEND_API_KEY` | from resend.com |
| `RESEND_FROM_EMAIL` | `noreply@yourdomain.com` |
| `TWILIO_ACCOUNT_SID` | from twilio.com |
| `TWILIO_AUTH_TOKEN` | from twilio.com |
| `TWILIO_FROM_NUMBER` | E.164 format |
| `STRIPE_SECRET_KEY` | from stripe.com |
| `STRIPE_WEBHOOK_SECRET` | from Stripe webhook endpoint |
| `STRIPE_PRICE_PRO` | Stripe Price ID for Pro plan |
| `STRIPE_PRICE_CHAIN` | Stripe Price ID for Chain plan |
| `APP_URL` | `https://${{ Frontend.RAILWAY_PUBLIC_DOMAIN }}` |

### Frontend service

| Variable | Value |
|---|---|
| `NEXT_PUBLIC_API_URL` | `https://${{ Backend.RAILWAY_PUBLIC_DOMAIN }}` |
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | from stripe.com |
| `NODE_ENV` | `production` |

> Use Railway's `${{ Service.VARIABLE }}` reference syntax for inter-service refs — never paste resolved URLs.

---

## Stripe Setup

### Create products in Stripe

1. Go to https://dashboard.stripe.com/products
2. Create **PharmaSense Pro** — €29/month recurring. Copy the Price ID.
3. Create **PharmaSense Chain** — €99/month recurring. Copy the Price ID.
4. Set `STRIPE_PRICE_PRO` and `STRIPE_PRICE_CHAIN` in Railway backend env vars.

### Webhook

The webhook endpoint is `POST /api/v1/billing/webhook`. It handles:
- `checkout.session.completed` — activates subscription
- `customer.subscription.updated` — syncs status
- `customer.subscription.deleted` — downgrades to free
- `invoice.payment_failed` — marks as past_due

Signature verification uses `STRIPE_WEBHOOK_SECRET`. Events are idempotent via the `stripe_events` table.

---

## Architecture

```
pharmasense/
├── backend/              # Go 1.23 + Gin
│   ├── cmd/api/          # Server entrypoint
│   ├── cmd/seed/         # Demo data seeder
│   ├── internal/
│   │   ├── config/       # Env var config
│   │   ├── domain/       # Models
│   │   ├── db/           # Connection pool + migrations
│   │   ├── handlers/     # HTTP handlers
│   │   ├── middleware/   # JWT, plan enforcement, rate limiting
│   │   └── services/     # Risk engine, auth, notifications, billing
│   ├── migrations/       # SQL migration files
│   ├── railway.toml      # Railway config
│   └── railpack.json     # Railpack build config
│
├── frontend/             # Next.js 15 App Router
│   ├── src/app/          # Pages (locale-based routing)
│   ├── src/lib/          # API client, hooks, utils
│   ├── src/types/        # TypeScript types
│   └── src/messages/     # i18n (en.json, el.json)
│
├── docker-compose.yml    # Postgres 18 only
├── Makefile
└── README.md
```

### Risk Engine

Located in `internal/services/risk_engine.go`. Calculates risk for each inventory batch:

```
days_until_expiry = expiry_date - today
expected_sales    = avg_daily_sales × days_until_expiry  (rolling 90-day window)
surplus           = current_quantity - expected_sales

CRITICAL: days_until_expiry <= 30  AND surplus > 0
HIGH:     days_until_expiry <= 90  AND surplus > expected_sales × 0.5
MEDIUM:   days_until_expiry <= 180 AND surplus > expected_sales × 0.3
LOW:      otherwise
```

### Multi-Tenancy

- `pharmacy_id` is set from JWT claims only — never from request body or URL parameters.
- Pharmacy switching issues a new JWT via `POST /api/v1/pharmacies/switch`.
- Chain admins can switch between all pharmacies in their chain.

### Mock Services

All external services work without real API keys in development:
- **Email**: logs to stdout with `[MOCK EMAIL]` prefix
- **SMS**: logs to stdout with `[MOCK SMS]` prefix
- **Stripe**: returns mock URLs at `/mock-stripe/checkout` and `/mock-stripe/portal`

---

## Features

- **Expiry Monitoring** — real-time risk scoring for all inventory batches
- **Multi-Tenant** — each pharmacy is isolated; chain admins manage multiple pharmacies
- **Dashboard** — KPI cards, expiry timeline chart, top-risk product chart
- **Alerts** — per-batch actions: apply discount, transfer, return, dismiss
- **Reports** — savings over time, waste trend, category breakdown
- **CSV Import** — bulk upload with validation and error summary (Pro plan)
- **Billing** — Stripe subscriptions with 3 tiers: Free / Pro / Chain
- **Email** — Resend integration with welcome, verification, digest, reset emails
- **SMS** — Twilio integration for critical alerts (Pro plan, opt-in)
- **Bilingual** — English and Greek via next-intl
- **Mobile Responsive** — Tailwind CSS, tested at 375px

---

## License

MIT
