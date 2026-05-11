# PharmaSense — Claude Code Build Prompt (Railway-First, Full Features)

Build a complete, production-quality multi-tenant SaaS called **PharmaSense** — an expiry monitoring system for pharmacies in Cyprus. Follow every instruction below exactly. **Railway deployment is the #1 priority** — design every line of code so deploying to Railway is a 5-minute, zero-refactor process.

---

## 🎯 Project Overview

**PharmaSense** is a multi-tenant SaaS for pharmacies in Cyprus that monitors product expiry dates and helps reduce waste. Pharmacy owners sign up, add their staff and inventory, receive email/SMS alerts about expiring stock, and pay a monthly subscription.

**Core value:** "Save €1000s monthly by catching at-risk products before they expire"

**Languages:** English + Greek (bilingual via next-intl)

---

## 🛠️ Tech Stack (USE EXACTLY THESE)

### Backend
- **Go 1.22+** with **Gin** (`github.com/gin-gonic/gin`) — NOT Chi, NOT Echo
- **CORS:** `github.com/gin-contrib/cors`
- **PostgreSQL 18** — locally via **Docker** (single container, just for the DB), on Railway via Postgres 18 template
- **pgAdmin 4** — native desktop app for DB inspection (NOT in Docker)
- **DB driver:** `github.com/jackc/pgx/v5` + `pgxpool`
- **SQL:** `sqlc`
- **Migrations:** `github.com/golang-migrate/migrate/v4` — run programmatically on app startup
- **Validation:** Gin's `binding` tags via `go-playground/validator/v10`
- **Auth:** JWT (`github.com/golang-jwt/jwt/v5`)
- **Logging:** `log/slog` JSON handler to stdout
- **Config:** `github.com/joho/godotenv` for local dev
- **UUID:** PostgreSQL 18 native `uuidv7()` (time-ordered) + `github.com/google/uuid` in Go
- **Password hashing:** `golang.org/x/crypto/bcrypt` cost 12

### External Services (Production)
- **Email:** [Resend](https://resend.com) (`github.com/resend/resend-go/v2`) — generous free tier, simple API
- **SMS:** [Twilio](https://twilio.com) (`github.com/twilio/twilio-go`)
- **Payments:** [Stripe](https://stripe.com) (`github.com/stripe/stripe-go/v79`) — subscriptions + webhooks
- **All three MUST have a local "mock mode"** — if the API key env var is empty or starts with `mock_`, the service logs the action to stdout instead of calling the real API. This means local dev works with zero external accounts. Production on Railway just needs the real keys set.

### Frontend (ALL Tailwind CSS, no other styling)
- **Next.js 15** App Router with **TypeScript** strict mode
- **Tailwind CSS v4** — every component styled with Tailwind utility classes ONLY. NO inline styles, NO CSS modules, NO styled-components, NO emotion. Tailwind everywhere.
- **shadcn/ui** — installed via CLI (these components are themselves Tailwind, so this is fine)
- **react-hook-form + zod** for forms
- **TanStack Query v5** for data fetching
- **TanStack Table v8** for tables
- **Recharts** for charts
- **Zustand** only if truly needed
- **next-intl** for English + Greek
- **lucide-react** for icons
- **date-fns** for dates
- **@stripe/stripe-js + @stripe/react-stripe-js** for checkout

### DevOps
- **Docker** allowed BUT only for the local PostgreSQL container — nothing else.
- **Railway** as the deployment target — uses **Railpack** (replaced Nixpacks March 2026). Auto-detects Go and Next.js.
- **Makefile** for all local commands.

---

## 📊 Features

### 1. User Self-Registration (Multi-Tenant Signup Flow)

- `/signup` page collects: pharmacy name, license number, city, owner's full name, email, password.
- On submit: create a new `pharmacies` row + an admin `users` row in one transaction.
- Send a welcome email via Resend with a verification link.
- Auto-login after signup. Redirect to `/dashboard`.
- Email verification token in `users.email_verification_token` (nullable). Unverified accounts can use the app but show a banner: "Verify your email to unlock all features."
- Demo seeded accounts (admin@pharmasense.cy / Demo1234!) work alongside self-registered ones.

### 2. Multi-Pharmacy Chains

- A `chains` table groups multiple pharmacies under one organization.
- A pharmacy chain has multiple `pharmacies`, each with its own staff and inventory.
- A user has a `default_pharmacy_id` but can have access to multiple pharmacies in the same chain via a `pharmacy_users` join table with a role per pharmacy.
- Chain owners see a **pharmacy switcher** in the header — clicking switches the active pharmacy context for all subsequent API calls.
- A "Chain Dashboard" shows aggregated KPIs across all pharmacies in the chain (only for users with `chain_admin` role).

### 3. Expired Product Monitoring (THE CORE)

**Risk Calculation Engine** — `internal/services/risk_engine.go`, fully unit-tested:

```
days_until_expiry = expiry_date - today
expected_sales    = avg_daily_sales × days_until_expiry  (rolling 90-day window)
surplus           = current_quantity - expected_sales

risk_level:
  CRITICAL: days_until_expiry <= 30  AND surplus > 0
  HIGH:     days_until_expiry <= 90  AND surplus > expected_sales × 0.5
  MEDIUM:   days_until_expiry <= 180 AND surplus > expected_sales × 0.3
  LOW:      otherwise

estimated_loss     = surplus × purchase_price   (when HIGH or CRITICAL)
suggested_discount:
  CRITICAL: 30–50% / HIGH: 15–25% / MEDIUM: 10%
```

**Dashboard:** 4 KPI cards (critical count, estimated loss €, potential savings €, total inventory value €) + 2 charts (expiry timeline next 12 months, top 10 at-risk products) + recent alerts list.

**Inventory:** sortable/filterable table with risk-level color coding, bulk actions, CSV export.

**Alerts:** tabbed by risk level, per-card actions (apply discount, transfer, return, dismiss).

**Reports:** money saved over time, waste trend, action effectiveness, problematic categories.

**CSV Import:** drag-and-drop, validation preview, progress, summary.

### 4. Email Notifications (Resend)

Backend service `internal/services/notifications/email.go`:
- Welcome email on signup
- Email verification
- Password reset (request token → email link → reset page)
- **Daily digest** (8 AM Cyprus time) of CRITICAL items — triggered by a cron job (use `github.com/go-co-op/gocron/v2`)
- Subscription receipts (Stripe webhook → email)

In dev mode (empty API key), log emails to stdout with a clear `[MOCK EMAIL]` prefix.

### 5. SMS Notifications (Twilio)

Backend service `internal/services/notifications/sms.go`:
- Opt-in per user (`users.sms_enabled` boolean, `users.phone_number`)
- SMS for CRITICAL alerts only (high-value, time-sensitive products like prescription meds expiring within 7 days)
- Settings page lets user toggle SMS on/off and update phone number (with E.164 format validation)

In dev mode, log to stdout with `[MOCK SMS]`.

### 6. Payment Processing (Stripe Subscriptions)

3 plans (configured in Stripe dashboard, IDs in env vars):

| Plan | Price | Limits |
|---|---|---|
| **Free** | €0 | 1 pharmacy, 100 inventory items, no SMS, no daily digest |
| **Pro** | €29/mo | 1 pharmacy, unlimited inventory, SMS, digest |
| **Chain** | €99/mo | Up to 10 pharmacies, all features |

Implementation:
- `pharmacies.plan` column (`free` / `pro` / `chain`) and `pharmacies.stripe_customer_id`, `pharmacies.stripe_subscription_id`, `pharmacies.subscription_status`, `pharmacies.subscription_current_period_end`.
- `/billing` page shows current plan, "Upgrade" buttons that hit `POST /api/v1/billing/checkout-session` → returns a Stripe Checkout URL → redirect.
- `/api/v1/billing/portal-session` returns a Stripe Customer Portal URL for cancellation/payment method updates.
- `POST /api/v1/billing/webhook` handles Stripe events (`checkout.session.completed`, `customer.subscription.updated`, `customer.subscription.deleted`, `invoice.payment_failed`) — verify signature with `STRIPE_WEBHOOK_SECRET`.
- Middleware `RequirePlan("pro")` returns 402 Payment Required for features above the user's tier.
- In dev mode with empty Stripe keys, all checkout/portal endpoints return a mock URL like `/mock-stripe/checkout`, and a /mock-stripe page lets you simulate plan changes.

---

## 🗄️ Database Schema (PostgreSQL 18)

Use `uuidv7()` everywhere — time-ordered UUIDs give better index locality than v4.

`migrations/000001_init.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Pharmacy chains (parent of pharmacies)
CREATE TABLE chains (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    owner_email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE pharmacies (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    chain_id UUID REFERENCES chains(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    license_number VARCHAR(100) UNIQUE NOT NULL,
    address TEXT,
    city VARCHAR(100),
    phone VARCHAR(50),
    email VARCHAR(255),
    language VARCHAR(10) DEFAULT 'en',
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    subscription_status VARCHAR(50),
    subscription_current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_pharmacies_chain ON pharmacies(chain_id);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    default_pharmacy_id UUID REFERENCES pharmacies(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20),
    sms_enabled BOOLEAN DEFAULT FALSE,
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    password_reset_token VARCHAR(255),
    password_reset_expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Per-pharmacy role (a user may access multiple pharmacies in a chain)
CREATE TABLE pharmacy_users (
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'staff',  -- 'chain_admin' | 'admin' | 'staff'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (pharmacy_id, user_id)
);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    barcode VARCHAR(100) UNIQUE,
    name VARCHAR(500) NOT NULL,
    name_el VARCHAR(500),
    category VARCHAR(100),
    manufacturer VARCHAR(255),
    requires_prescription BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE inventory_batches (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    batch_number VARCHAR(100),
    expiry_date DATE NOT NULL,
    initial_quantity INTEGER NOT NULL,
    current_quantity INTEGER NOT NULL,
    purchase_price DECIMAL(10,2) NOT NULL,
    selling_price DECIMAL(10,2) NOT NULL,
    supplier VARCHAR(255),
    received_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_batches_pharmacy_expiry ON inventory_batches(pharmacy_id, expiry_date);
CREATE INDEX idx_batches_product ON inventory_batches(product_id);

CREATE TABLE sales (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    batch_id UUID NOT NULL REFERENCES inventory_batches(id),
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    sale_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_sales_pharmacy_date ON sales(pharmacy_id, sale_date);
CREATE INDEX idx_sales_product ON sales(product_id, sale_date);

CREATE TABLE risk_assessments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    batch_id UUID NOT NULL REFERENCES inventory_batches(id) ON DELETE CASCADE,
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id) ON DELETE CASCADE,
    risk_level VARCHAR(20) NOT NULL,
    days_until_expiry INTEGER NOT NULL,
    avg_daily_sales DECIMAL(10,2),
    expected_sales INTEGER,
    estimated_surplus INTEGER,
    estimated_loss DECIMAL(10,2),
    suggested_discount_percent INTEGER,
    calculated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_risk_pharmacy_level ON risk_assessments(pharmacy_id, risk_level);

CREATE TABLE alert_actions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    batch_id UUID NOT NULL REFERENCES inventory_batches(id),
    pharmacy_id UUID NOT NULL REFERENCES pharmacies(id),
    user_id UUID NOT NULL REFERENCES users(id),
    action_type VARCHAR(50) NOT NULL,
    discount_percent INTEGER,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Log of every email/SMS sent (for debugging and audit)
CREATE TABLE notification_log (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id),
    pharmacy_id UUID REFERENCES pharmacies(id),
    channel VARCHAR(20) NOT NULL,    -- 'email' | 'sms'
    template VARCHAR(100) NOT NULL,  -- 'welcome' | 'daily_digest' | 'critical_alert' | ...
    recipient VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,     -- 'sent' | 'failed' | 'mocked'
    error_message TEXT,
    sent_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_notif_log_pharmacy ON notification_log(pharmacy_id, sent_at DESC);

-- Stripe webhook event log (idempotency)
CREATE TABLE stripe_events (
    id VARCHAR(255) PRIMARY KEY,     -- Stripe event ID
    type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
```

Plus a corresponding `000001_init.down.sql` that drops everything in reverse order.

---

## 🌱 Demo Data (`cmd/seed/main.go`)

Idempotent (wipes and re-seeds).

### Demo Chain + Pharmacies
- Chain: "Nicosia Health Group"
- 3 pharmacies in the chain:
  - "Nicosia Central Pharmacy" (Nicosia) — `CY-PH-2024-001`
  - "Limassol Marina Pharmacy" (Limassol) — `CY-PH-2024-002`
  - "Paphos Tourist Pharmacy" (Paphos) — `CY-PH-2024-003`

### Demo Users (bcrypt cost 12)
- `chain_admin@pharmasense.cy` / `Demo1234!` — `chain_admin` role across all 3
- `admin@pharmasense.cy` / `Demo1234!` — `admin` of Nicosia Central
- `staff@pharmasense.cy` / `Demo1234!` — `staff` of Nicosia Central

All seeded users have `email_verified = TRUE`.

### Plans
- Nicosia Central: `pro` (active subscription, mock Stripe IDs)
- Limassol Marina: `free`
- Paphos Tourist: `chain` (under the chain plan)

### ~150 Products, ~500 Batches per Pharmacy, ~10,000 Sales (last 90 days)

Distribution per pharmacy: 20% CRITICAL / 25% HIGH / 20% MEDIUM / 35% LOW.

Realistic data — see categories below. NEVER "Product 1, Product 2".

Categories: Painkillers (Paracetamol, Ibuprofen, Aspirin, …), Antibiotics, Vitamins, Cold/Flu, Allergy, Digestive, Cardiovascular, Diabetes, Skincare, Baby care, First aid. Manufacturers: Pfizer, GSK, Bayer, Sanofi, Roche. Suppliers: "MedSupply Cyprus", "PharmaWholesale Ltd", "EuroMeds".

Run risk engine after seeding so dashboards show real numbers immediately.

---

## 🔌 API Endpoints

All under `/api/v1`. JWT required except `/auth/*`, `/healthz`, `/billing/webhook`.

```
GET  /healthz                                # Railway health check

POST /auth/signup                            # Self-registration
POST /auth/login
POST /auth/logout
GET  /auth/me
POST /auth/verify-email                      # via token
POST /auth/forgot-password
POST /auth/reset-password                    # via token

GET    /pharmacies                           # All pharmacies user has access to
GET    /pharmacies/:id
PATCH  /pharmacies/:id
POST   /pharmacies/switch                    # Switch active pharmacy context

GET    /chains/:id                           # Chain admin only
GET    /chains/:id/dashboard                 # Aggregated KPIs across pharmacies

GET    /products
GET    /products/:id
POST   /products
PATCH  /products/:id

GET    /inventory
GET    /inventory/:id
POST   /inventory
PATCH  /inventory/:id
DELETE /inventory/:id
POST   /inventory/import                     # CSV multipart

GET    /sales
POST   /sales
GET    /sales/stats

GET    /risk/dashboard
GET    /risk/assessments
POST   /risk/recalculate
GET    /risk/timeline

GET    /alerts
POST   /alerts/:batch_id/action

GET    /reports/savings
GET    /reports/waste
GET    /reports/categories

GET    /settings/notifications               # Per-user email/SMS prefs
PATCH  /settings/notifications

POST   /billing/checkout-session             # Returns Stripe Checkout URL
POST   /billing/portal-session               # Returns Stripe Customer Portal URL
GET    /billing/subscription                 # Current plan + status
POST   /billing/webhook                      # Stripe webhook (signature-verified, unauthenticated)
```

Multi-tenancy enforced everywhere: `pharmacy_id` from JWT claims, NEVER from request body/URL. The active pharmacy in the JWT changes only via `/pharmacies/switch` (which verifies the user has access).

---

## ⚙️ Gin Implementation Notes

```go
// server bootstrap
r := gin.New()
r.Use(gin.Recovery())
r.Use(middleware.SlogLogger())
r.Use(cors.New(corsConfig))

api := r.Group("/api/v1")
api.GET("/healthz", h.Healthz)
api.POST("/auth/signup", authH.Signup)
api.POST("/auth/login", authH.Login)
api.POST("/billing/webhook", billingH.Webhook)  // unauthenticated, signature-verified

protected := api.Group("/")
protected.Use(middleware.JWTAuth(cfg.JWTSecret))
protected.GET("/risk/dashboard", riskH.Dashboard)
// ...

pro := protected.Group("/")
pro.Use(middleware.RequirePlan("pro"))
pro.POST("/inventory/import", invH.Import)  // CSV import is Pro-only
```

Binding example:
```go
type SignupRequest struct {
    PharmacyName   string `json:"pharmacy_name"   binding:"required,min=2"`
    LicenseNumber  string `json:"license_number"  binding:"required"`
    City           string `json:"city"            binding:"required"`
    FullName       string `json:"full_name"       binding:"required"`
    Email          string `json:"email"           binding:"required,email"`
    Password       string `json:"password"        binding:"required,min=8"`
}
```

---

## 🚂 RAILWAY DEPLOYMENT — THE #1 PRIORITY

Every line of code must be Railway-ready from day one.

### Backend Rules

1. **Listen on `$PORT` and `0.0.0.0`**:
   ```go
   port := os.Getenv("PORT")
   if port == "" { port = "3001" }
   srv := &http.Server{ Addr: "0.0.0.0:" + port, Handler: r }
   ```

2. **`DATABASE_URL` from env**, never hardcoded. Railway injects it when you link the Postgres service.

3. **CORS origin from env** — `FRONTEND_ORIGIN`.

4. **Health check** `GET /healthz` returns 200 + `{"status":"ok"}`. **Without this, Railway deploys hang.**

5. **Auto-run migrations on startup**. Before `srv.ListenAndServe()`:
   ```go
   if err := db.RunMigrations(cfg.DatabaseURL); err != nil { log.Fatal(err) }
   ```
   Use `golang-migrate` with `file://migrations` source.

6. **Graceful shutdown** on `SIGTERM`:
   ```go
   quit := make(chan os.Signal, 1)
   signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
   <-quit
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()
   srv.Shutdown(ctx)
   ```

7. **slog JSON to stdout** — Railway captures it automatically.

8. **Cron job** (`gocron`) must respect a `DISABLE_CRON=true` env var so you can scale to multiple instances later without duplicate sends.

### Frontend Rules

1. **API URL from env**:
   ```ts
   // lib/api/client.ts
   const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001";
   ```
   **No `localhost:3001` anywhere else in frontend code.**

2. **`output: 'standalone'`** in `next.config.js`:
   ```js
   module.exports = {
       output: 'standalone',
       experimental: { trustHostHeader: true },
   };
   ```

3. **Start uses `$PORT`** in `package.json`:
   ```json
   "start": "next start -p ${PORT:-3000}"
   ```

### Railway Config Files (commit these)

**`backend/railway.toml`:**
```toml
[build]
builder = "RAILPACK"

[deploy]
startCommand = "./bin/api"
healthcheckPath = "/healthz"
healthcheckTimeout = 30
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 3
```

**`backend/railpack.json`:**
```json
{
  "$schema": "https://schema.railpack.com",
  "provider": "go",
  "steps": {
    "install": { "commands": ["go mod download"] },
    "build":   { "commands": ["go build -o bin/api ./cmd/api"] }
  },
  "deploy": { "startCommand": "./bin/api" }
}
```

**`frontend/railway.toml`:**
```toml
[build]
builder = "RAILPACK"

[deploy]
startCommand = "npm run start"
healthcheckPath = "/"
healthcheckTimeout = 30
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 3
```

### Railway Environment Variables

**Backend service:**
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
| `STRIPE_WEBHOOK_SECRET` | from stripe webhook endpoint |
| `STRIPE_PRICE_PRO` | Stripe Price ID for Pro plan |
| `STRIPE_PRICE_CHAIN` | Stripe Price ID for Chain plan |
| `APP_URL` | `https://${{ Frontend.RAILWAY_PUBLIC_DOMAIN }}` |

**Frontend service:**
| Variable | Value |
|---|---|
| `NEXT_PUBLIC_API_URL` | `https://${{ Backend.RAILWAY_PUBLIC_DOMAIN }}` |
| `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | from stripe.com |
| `NODE_ENV` | `production` |

Use Railway's `${{ Service.VARIABLE }}` reference syntax for inter-service refs — never paste resolved URLs.

### Step-by-Step Railway Deploy (include verbatim in README)

```markdown
## Deploy to Railway

1. **Push to GitHub.** Create a repo and push the code.

2. **Create a Railway project** at https://railway.com → New Project → Empty Project → name it `pharmasense`.

3. **Add PostgreSQL 18.**
   - + Create → Database → PostgreSQL.
   - Make sure it's version 18. If the default is older, use the template: https://railway.com/deploy/postgres-18-ssl
   - Railway provisions it and creates `DATABASE_URL`.

4. **Add Backend service.**
   - + Create → GitHub Repo → select your repo.
   - Settings → Root Directory: `/backend`
   - Settings → Watch Paths: `/backend/**`
   - Variables: add all backend vars from the table above. Use `${{ Postgres.DATABASE_URL }}` syntax for the DB.
   - Settings → Networking → Generate Domain.

5. **Add Frontend service.**
   - + Create → GitHub Repo → same repo.
   - Settings → Root Directory: `/frontend`
   - Settings → Watch Paths: `/frontend/**`
   - Variables: `NEXT_PUBLIC_API_URL = https://${{ Backend.RAILWAY_PUBLIC_DOMAIN }}`, plus Stripe publishable key.
   - Networking → Generate Domain.

6. **Configure Stripe webhook.**
   - In the Stripe dashboard, add a webhook endpoint pointing to `https://<backend-domain>/api/v1/billing/webhook`.
   - Copy the signing secret → set `STRIPE_WEBHOOK_SECRET` on the backend.

7. **Wait for all three services to be green.** Backend auto-migrates on startup.

8. **Seed demo data** (optional, one-time):
   ```bash
   npm install -g @railway/cli
   railway login
   railway link
   railway run --service backend go run ./cmd/seed
   ```

9. **Open the frontend domain** → log in with `admin@pharmasense.cy` / `Demo1234!` or sign up a new account.
```

---

## 🐳 Local PostgreSQL via Docker (DB ONLY)

The user wants Docker only for the local Postgres. Single-purpose `docker-compose.yml` at repo root:

```yaml
services:
  postgres:
    image: postgres:18-alpine
    container_name: pharmasense-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: pharmasense
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d pharmasense"]
      interval: 5s
      timeout: 5s
      retries: 10

volumes:
  pgdata:
```

The backend and frontend run **natively** with `make dev-backend` / `make dev-frontend` — they are NOT in Docker. Only Postgres is.

### Connecting pgAdmin 4 to the Docker Postgres

README must include this walkthrough:

1. Install pgAdmin 4 (macOS: `brew install --cask pgadmin4`, Linux: see pgadmin.org, Windows: bundled with Postgres installer).
2. Open pgAdmin 4 → set master password (first launch).
3. Right-click **Servers** → Register → Server.
4. **General** → Name: `PharmaSense Local`.
5. **Connection** tab:
   - Host: `localhost`
   - Port: `5432`
   - Maintenance database: `pharmasense`
   - Username: `postgres`
   - Password: `postgres` (check "Save Password")
6. Save → expand server → Databases → pharmasense → Schemas → public → Tables to inspect rows.

### Connecting pgAdmin 4 to Railway's Postgres (production)

1. Railway dashboard → Postgres service → Variables tab.
2. Find `DATABASE_PUBLIC_URL`. Parse: `postgresql://user:password@host:port/database`.
3. pgAdmin → Register Server. Fill from URL.
4. **SSL tab → SSL mode: `require`.**
5. Save and inspect production data.

---

## 📁 Project Structure

```
pharmasense/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go              # Gin entrypoint
│   │   └── seed/main.go             # Idempotent seed
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── handlers/
│   │   │   ├── auth.go              # signup, login, verify, reset
│   │   │   ├── billing.go           # Stripe checkout, portal, webhook
│   │   │   ├── chains.go
│   │   │   ├── pharmacies.go
│   │   │   ├── inventory.go
│   │   │   ├── alerts.go
│   │   │   ├── risk.go
│   │   │   ├── reports.go
│   │   │   └── settings.go
│   │   ├── middleware/
│   │   │   ├── auth.go              # JWT
│   │   │   ├── plan.go              # RequirePlan("pro")
│   │   │   ├── logger.go            # slog
│   │   │   └── ratelimit.go
│   │   ├── services/
│   │   │   ├── risk_engine.go       # + risk_engine_test.go
│   │   │   ├── auth_service.go
│   │   │   ├── billing/
│   │   │   │   ├── stripe.go        # real impl
│   │   │   │   └── mock.go          # mock when keys empty
│   │   │   ├── notifications/
│   │   │   │   ├── email.go         # Resend + mock
│   │   │   │   ├── sms.go           # Twilio + mock
│   │   │   │   └── templates.go
│   │   │   └── cron/
│   │   │       └── daily_digest.go
│   │   ├── repository/              # sqlc-generated
│   │   ├── db/
│   │   │   └── migrations.go        # programmatic migration runner
│   │   └── server/server.go         # Gin engine + routes
│   ├── migrations/
│   │   ├── 000001_init.up.sql
│   │   └── 000001_init.down.sql
│   ├── queries/                     # sqlc input
│   ├── sqlc.yaml
│   ├── go.mod
│   ├── .env.example
│   ├── railway.toml
│   └── railpack.json
│
├── frontend/
│   ├── src/
│   │   ├── app/[locale]/
│   │   │   ├── (auth)/
│   │   │   │   ├── login/page.tsx
│   │   │   │   ├── signup/page.tsx
│   │   │   │   ├── forgot-password/page.tsx
│   │   │   │   └── verify-email/page.tsx
│   │   │   ├── (dashboard)/
│   │   │   │   ├── dashboard/page.tsx
│   │   │   │   ├── inventory/page.tsx
│   │   │   │   ├── alerts/page.tsx
│   │   │   │   ├── import/page.tsx
│   │   │   │   ├── reports/page.tsx
│   │   │   │   ├── billing/page.tsx
│   │   │   │   ├── chain/page.tsx       # chain admins only
│   │   │   │   └── settings/page.tsx
│   │   │   └── layout.tsx
│   │   ├── components/{ui,layout,dashboard,inventory,alerts,billing}/
│   │   ├── lib/{api,hooks,utils,stripe}/
│   │   ├── types/
│   │   └── messages/{en.json,el.json}
│   ├── package.json
│   ├── next.config.js                  # output: 'standalone'
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   └── railway.toml
│
├── docker-compose.yml                   # Postgres ONLY
├── Makefile
├── README.md
├── PROGRESS.md                          # Created last
└── .gitignore                           # .env, bin/, node_modules/, .next/
```

---

## 🚀 Makefile

```makefile
.PHONY: setup db-up db-down db-logs migrate-up migrate-down seed sqlc dev-backend dev-frontend test build

DB_URL ?= postgres://postgres:postgres@localhost:5432/pharmasense?sslmode=disable

setup:
	cd backend && go mod download
	cd frontend && npm install

db-up:
	docker compose up -d postgres
	@echo "Waiting for Postgres..."
	@until docker exec pharmasense-db pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
	@echo "Postgres ready on localhost:5432"

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

migrate-up:
	migrate -path backend/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path backend/migrations -database "$(DB_URL)" down

sqlc:
	cd backend && sqlc generate

seed:
	cd backend && go run ./cmd/seed

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend:
	cd frontend && npm run dev

test:
	cd backend && go test ./...

build:
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && npm run build
```

3-command local startup: `make setup && make db-up migrate-up seed`, then `make dev-backend` + `make dev-frontend` in separate terminals.

---

## 🎨 UI/UX (ALL Tailwind CSS)

**Every styled element uses Tailwind utility classes.** No CSS modules, no inline styles, no styled-components.

- **Primary:** `emerald-600` (trust, health)
- **Risk badges:** `rounded-full px-3 py-1 text-xs font-semibold` with bg/text:
  - CRITICAL: `bg-red-100 text-red-700`
  - HIGH: `bg-orange-100 text-orange-700`
  - MEDIUM: `bg-yellow-100 text-yellow-800`
  - LOW: `bg-green-100 text-green-700`
- **Cards:** `rounded-xl border border-slate-200 bg-white shadow-sm`
- **Buttons:** `inline-flex items-center gap-2 rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-emerald-700 disabled:opacity-50`
- Sidebar nav, collapsible on mobile via Tailwind responsive utilities.
- Top header: pharmacy switcher dropdown, language switcher (EN ↔ EL), user menu, plan badge (Free / Pro / Chain).
- shadcn `Skeleton` for loading, `Sonner` for toasts.
- Empty states with friendly Tailwind-styled messages.
- Mobile-responsive — test at 375px.
- WCAG AA accessibility.

---

## ✅ Acceptance Criteria

The build is done when ALL pass:

### Local
1. `make setup && make db-up migrate-up seed` runs cleanly.
2. `make dev-backend` starts Gin on `:3001`. `curl localhost:3001/healthz` returns 200.
3. `make dev-frontend` starts Next.js on `:3000`.
4. Login works with seeded demo creds.
5. Signup creates a new pharmacy + user, sends welcome email (logged in dev), auto-logs-in.
6. Dashboard shows realistic non-zero numbers.
7. Inventory has 500+ batches per pharmacy with varied risks.
8. Alerts page shows CRITICAL items with working action buttons.
9. Pharmacy switcher works for chain_admin user (cycles across 3 pharmacies).
10. Chain dashboard shows aggregated KPIs for chain_admin.
11. Billing page shows current plan, "Upgrade" returns mock Stripe URL in dev.
12. Settings: toggle SMS on, save phone, trigger a CRITICAL alert → `[MOCK SMS]` line in backend logs.
13. Language switcher works (EN ↔ EL) — all UI translates.
14. Mobile responsive at 375px.
15. CSV import works with validation.
16. Multi-tenancy enforced: a JWT for pharmacy A returns 403/empty for pharmacy B's data.
17. `risk_engine_test.go` covers all 4 levels.
18. No console errors.

### Railway-Ready
19. `PORT`, `DATABASE_URL`, `JWT_SECRET`, `FRONTEND_ORIGIN`, `NEXT_PUBLIC_API_URL`, Stripe/Resend/Twilio keys — ALL from env.
20. Backend listens on `0.0.0.0:$PORT`.
21. Migrations run automatically on startup.
22. `GET /healthz` returns 200.
23. Frontend `next.config.js` has `output: 'standalone'`.
24. `backend/railway.toml`, `backend/railpack.json`, `frontend/railway.toml` exist.
25. Stripe webhook endpoint verifies signature before processing.
26. No `localhost` strings in frontend outside env fallbacks.
27. `.env` is gitignored; `.env.example` is committed with placeholder mock values.
28. README has complete sections: Local Setup with Docker Postgres, pgAdmin 4 walkthrough, Deploy to Railway, Stripe webhook setup.

### Production-Quality
29. UI is beautiful — would not embarrass you in front of a real pharmacy owner.
30. Demo data is realistic (no "Product 1, Product 2").
31. Email/SMS/Stripe all have mock fallbacks for local dev.

---

## 📋 PROGRESS.md (create LAST)

Create `PROGRESS.md` at repo root with this template, filled in honestly:

```markdown
# PharmaSense — Build Progress

Last updated: <date>

## ✅ Completed
- [ ] Docker Compose for local Postgres 18
- [ ] Backend: schema + migrations with uuidv7
- [ ] Backend: sqlc setup + queries
- [ ] Backend: Gin server, CORS, slog, /healthz
- [ ] Backend: JWT auth (signup, login, verify, reset)
- [ ] Backend: risk engine + unit tests
- [ ] Backend: all CRUD handlers
- [ ] Backend: Stripe billing + webhooks (with mock mode)
- [ ] Backend: Resend email (with mock mode)
- [ ] Backend: Twilio SMS (with mock mode)
- [ ] Backend: daily digest cron job
- [ ] Backend: multi-tenancy + RequirePlan middleware
- [ ] Backend: chain dashboard
- [ ] Backend: seed (chain + 3 pharmacies + users + data)
- [ ] Backend: auto-migration on startup
- [ ] Backend: graceful SIGTERM shutdown
- [ ] Backend: railway.toml + railpack.json
- [ ] Frontend: Next.js 15 + Tailwind v4 + shadcn
- [ ] Frontend: next-intl EN + EL
- [ ] Frontend: signup + login + verify-email + forgot/reset flows
- [ ] Frontend: protected dashboard layout with pharmacy switcher
- [ ] Frontend: dashboard with KPIs + Recharts
- [ ] Frontend: inventory + alerts + reports + import + settings
- [ ] Frontend: billing page with Stripe Checkout
- [ ] Frontend: chain dashboard (chain_admin only)
- [ ] Frontend: notification settings (SMS toggle, phone)
- [ ] Frontend: output: 'standalone' + railway.toml
- [ ] Makefile + docker-compose.yml
- [ ] README with all required sections
- [ ] .gitignore + .env.example

## 🚧 In Progress
(anything actively being worked on)

## 📋 Remaining
(anything not started)

## 🧠 Key Decisions & Context
- PostgreSQL 18 via Docker locally (single container, single purpose), Railway Postgres in production.
- pgAdmin 4 as desktop app, NOT in Docker.
- Backend and frontend run natively (not in Docker) both locally and on Railway.
- All external services (Stripe / Resend / Twilio) have mock implementations — local dev works with zero accounts.
- Migrations run programmatically on backend startup — no manual step on Railway.
- Multi-tenancy: `pharmacy_id` from JWT only, never from request payload.
- Chain admins switch active pharmacy via `/pharmacies/switch` → new JWT issued.
- Stripe webhook is unauthenticated but signature-verified.
- Daily digest cron uses gocron, disabled via `DISABLE_CRON=true` env var for scale-out.
- All UI is Tailwind utility classes (plus shadcn which is also Tailwind-based).
- Uses Postgres 18's native `uuidv7()` for time-ordered primary keys.

## 🔮 Next Session (if resuming)
1. Verify `make db-up && make dev-backend && make dev-frontend` works end-to-end.
2. Test signup flow → check `[MOCK EMAIL]` in backend logs.
3. Test demo login → verify dashboard numbers non-zero.
4. Test pharmacy switcher with chain_admin user.
5. Test billing with mock Stripe.
6. Push to GitHub.
7. Follow README's "Deploy to Railway" section.
8. Add real Stripe/Resend/Twilio keys in Railway dashboard.
9. Configure Stripe webhook URL pointing to Railway backend domain.

## 🐛 Known Issues / TODOs
(list anything half-done, hacks, or future improvements)
```

Mark each checkbox honestly — half-done items go under "In Progress", not "Completed".

---

## 🔒 Security

- bcrypt cost 12.
- JWT 24h expiry, secret from env, never logged.
- Stripe webhook signature verification (use `webhook.ConstructEvent`).
- Idempotent webhook processing (check `stripe_events` table before processing).
- SQL via sqlc only — no string concatenation.
- CORS locked to `FRONTEND_ORIGIN`.
- Rate limit `/auth/login` and `/auth/signup` (in-memory token bucket).
- `.env` gitignored, `.env.example` committed.
- HTTPS automatic on Railway.

---

## 💡 Implementation Order

1. Git init + `.gitignore` + `docker-compose.yml`.
2. Backend: `go mod`, deps, schema migration, sqlc.
3. Backend: config, programmatic migrate-on-startup, Gin server, `/healthz`.
4. Backend: JWT middleware, signup + login + verify + reset.
5. Backend: notification services with mock fallback (Resend + Twilio).
6. Backend: risk engine + tests.
7. Backend: inventory, alerts, dashboard, reports, chain, settings handlers.
8. Backend: Stripe billing (checkout, portal, webhook) with mock fallback.
9. Backend: daily digest cron.
10. Backend: seed script (chain + 3 pharmacies + users + data).
11. Backend: `railway.toml` + `railpack.json` + `.env.example`.
12. Frontend: `create-next-app`, Tailwind v4, shadcn CLI, next-intl.
13. Frontend: `lib/api/client.ts` reading `NEXT_PUBLIC_API_URL`.
14. Frontend: signup/login/verify/reset pages.
15. Frontend: protected layout + pharmacy switcher.
16. Frontend: dashboard with KPIs + charts.
17. Frontend: inventory + alerts + reports + import + settings + billing + chain.
18. Frontend: `next.config.js` standalone + `railway.toml`.
19. Top-level `Makefile`.
20. `README.md` with all sections (especially Deploy to Railway).
21. **`PROGRESS.md`** at the very end.

---

## 🎬 Final Instructions

- Clean, idiomatic Go and TypeScript.
- Comments only where logic is non-obvious — especially risk engine and Stripe webhook.
- Wrap errors with context: `fmt.Errorf("signup: create pharmacy: %w", err)`.
- **Make the UI beautiful** — this gets shown to real pharmacy owners.
- **Realistic demo data** — no "Product 1, Product 2".
- Test end-to-end before declaring done: signup → log in → see dashboard → switch pharmacy → upgrade plan (mock) → check alerts → switch language → resize to mobile.
- Do not ask for confirmation between steps — execute the full plan.
- Create `PROGRESS.md` LAST with accurate status.

**Quality > Quantity.** Polished multi-tenant SaaS that actually deploys to Railway in 5 minutes is the goal.

Begin now. Build backend → frontend → docs → `PROGRESS.md`.
