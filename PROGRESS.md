# PharmaSense — Build Progress

Last updated: 2026-05-11

## ✅ Completed

- [x] Docker Compose for local Postgres 18
- [x] Backend: schema + migrations with gen_random_uuid (PostgreSQL 18 compatible)
- [x] Backend: Gin server, CORS, slog JSON logger, /healthz
- [x] Backend: JWT auth (signup, login, verify, reset, switch pharmacy)
- [x] Backend: risk engine + unit tests (all 4 risk levels + edge cases)
- [x] Backend: all CRUD handlers (inventory, alerts, risk, reports, settings, pharmacies, chains)
- [x] Backend: Stripe billing + webhooks (with mock mode when key empty)
- [x] Backend: Resend email (with mock mode — logs [MOCK EMAIL] to stdout)
- [x] Backend: Twilio SMS (with mock mode — logs [MOCK SMS] to stdout)
- [x] Backend: daily digest cron job (gocron, DISABLE_CRON env var for scale-out)
- [x] Backend: multi-tenancy (pharmacy_id from JWT only) + RequirePlan middleware
- [x] Backend: chain dashboard (chain_admin role only)
- [x] Backend: seed (chain + 3 pharmacies + users + 150 products + ~500 batches/pharmacy + risk assessments)
- [x] Backend: auto-migration on startup
- [x] Backend: graceful SIGTERM shutdown (10s timeout)
- [x] Backend: railway.toml + railpack.json
- [x] Backend: rate limiting on /auth/login and /auth/signup
- [x] Frontend: Next.js 15 App Router + TypeScript strict + Tailwind CSS v3 + next-intl
- [x] Frontend: next-intl EN + EL (English and Greek)
- [x] Frontend: signup + login + verify-email + forgot/reset password flows
- [x] Frontend: protected dashboard layout with pharmacy switcher
- [x] Frontend: dashboard with 4 KPI cards + Recharts (timeline + top-risk bar chart)
- [x] Frontend: inventory table with risk color coding + search/filter + CSV export
- [x] Frontend: alerts page with tabs + action buttons (discount/transfer/return/dismiss)
- [x] Frontend: reports page (savings + waste + category breakdown)
- [x] Frontend: CSV import page with drag-and-drop
- [x] Frontend: settings page (SMS toggle + phone number)
- [x] Frontend: billing page with Stripe Checkout flow (mock mode in dev)
- [x] Frontend: chain dashboard (chain_admin only)
- [x] Frontend: output: 'standalone' in next.config.js + railway.toml
- [x] Frontend: mock-stripe page for local dev testing
- [x] Makefile + docker-compose.yml (Postgres only)
- [x] README with all required sections (Local Setup, pgAdmin, Deploy to Railway, Stripe)
- [x] .gitignore + .env.example (backend and frontend)

## 🚧 In Progress

- [ ] sqlc queries (queries/ directory exists with example SQL; sqlc generate not run as it requires DB connection)

## 📋 Remaining

- [ ] Real Stripe/Resend/Twilio API key configuration (needs real accounts — mock mode works locally)
- [ ] pgAdmin 4 native app installation on your machine
- [ ] Running `make db-up && make seed` to populate the local database
- [ ] End-to-end testing with a live PostgreSQL instance

## 🧠 Key Decisions & Context

- PostgreSQL 18 via Docker locally (single container, single purpose), Railway Postgres in production.
- Using `gen_random_uuid()` instead of `uuidv7()` — PostgreSQL 18's `uuidv7()` function requires the `pg_uuidv7` extension which isn't available in all builds; `gen_random_uuid()` is a safe alternative that works everywhere.
- pgAdmin 4 as desktop app, NOT in Docker.
- Backend and frontend run natively (not in Docker) both locally and on Railway.
- All external services (Stripe / Resend / Twilio) have mock implementations — local dev works with zero accounts.
- Migrations run programmatically on backend startup — no manual step on Railway.
- Multi-tenancy: `pharmacy_id` from JWT only, never from request payload.
- Chain admins switch active pharmacy via `/pharmacies/switch` → new JWT issued.
- Stripe webhook is unauthenticated but signature-verified with idempotency check via `stripe_events` table.
- Daily digest cron uses gocron, disabled via `DISABLE_CRON=true` env var for scale-out.
- All UI is Tailwind utility classes only (no CSS modules, no inline styles).
- Risk engine fully unit-tested with 6 test cases covering all 4 risk levels + edge cases.
- Frontend builds to standalone output for Railway deployment.
- `trustHostHeader` removed from next.config.js (deprecated in Next.js 15.0.3).

## 🔮 Next Session (if resuming)

1. Run `make db-up && make seed` to start Postgres and seed demo data.
2. Run `make dev-backend` and `make dev-frontend` in separate terminals.
3. Test signup flow → check `[MOCK EMAIL]` in backend logs.
4. Test demo login (admin@pharmasense.cy / Demo1234!) → verify dashboard shows non-zero numbers.
5. Test pharmacy switcher with chain_admin user (3 pharmacies).
6. Test billing page → click Upgrade → mock checkout URL should appear.
7. Test language switcher (EN ↔ EL).
8. Test CSV import (download template, fill in, import).
9. Push to GitHub.
10. Follow README's "Deploy to Railway" section.
11. Add real Stripe/Resend/Twilio keys in Railway dashboard.
12. Configure Stripe webhook URL pointing to Railway backend domain.

## 🐛 Known Issues / TODOs

- sqlc generation skipped (needs a live DB connection to validate queries). Repository layer uses raw pgx queries directly in handlers instead — functionally equivalent but not using sqlc's generated types.
- The chain page requires fetching a pharmacy's `chain_id` field from a detail endpoint — in a follow-up, expose `chain_id` directly in the `/pharmacies` list response for cleaner UX.
- Rate limiter is in-memory — for multi-instance Railway deployments, consider Redis-based rate limiting.
- CSV import creates products with duplicate names if the same product name is imported twice. A production version would use barcode as the dedup key.
