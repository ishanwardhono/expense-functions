# expense-functions

Go backend for a personal expense-tracking app (single user, no auth, `CORS: *`),
deployed as a **Google Cloud Function** (functions-framework-go) over **CockroachDB**.

This is the **v2 "Amplop" (envelope budgeting)** backend. The authoritative design
is [`docs/superpowers/specs/2026-06-15-amplop-v2-backend-design.md`](docs/superpowers/specs/2026-06-15-amplop-v2-backend-design.md).

## Architecture

Strict layering — **handler (HTTP) → service (orchestration, validation,
effective-date resolution) → repo (SQL)** — with a pure envelope engine.

```
function.go                  registers ONE HTTP function "Expense" → internal router
cmd/main.go                  local dev server (functions-framework StartHostPort)
internal/
  platform/  config · database(sqlx) · httpx(router, CORS, JSON, error→HTTP) · apierr · timeutil
  envelope/  PURE engine: EnvelopeOf + ComputeMonth over a resolved month context
  expense/   model · repo · service · handler   (all transactions, incl. subscription payments)
  subscription/  identity + effective-dated versions + resolution
  budget/    effective-dated config + resolution
  month/     resolve config+subs → run engine → assemble GET /month dashboard
migrations/  0001_init_amplop.sql
```

All envelope/budget math is server-side; the API returns render-ready integers and
the client formats. Timezone is **Asia/Jakarta**; the `TIME` env var (RFC3339)
pins "now" and the current month for deterministic tests and effective-dated writes.

## API

One routed function; JSON in/out. Errors are non-2xx `{"error":"message"}`
(400 validation, 404 unknown id, 409 duplicate subscription payment, 500 default).

| Method | Path | Purpose |
|--------|------|---------|
| GET    | `/month?year=&month=` | month dashboard (envelopes, weeks, weekends, calendar, days, subscriptions) |
| POST   | `/expenses` | create expense (incl. Langganan payment) |
| PUT    | `/expenses/{id}` | update expense |
| DELETE | `/expenses/{id}` | delete expense (204) |
| GET    | `/subscriptions?year=&month=` | resolved subscription set for the month |
| POST   | `/subscriptions` | create subscription (effective from current month) |
| PUT    | `/subscriptions/{id}` | update name/color (identity) and/or alloc/due_day (new version) |
| DELETE | `/subscriptions/{id}` | soft-end from current month |
| GET    | `/budget?year=&month=` | resolved budget config for the month |
| PUT    | `/budget` | upsert budget config effective from current month |

See spec §7 for the full request/response shapes.

## Run locally

The function talks to CockroachDB. For local dev, run an **insecure** single-node
CockroachDB in Docker (no certs; `DB_SSL_MODE=disable`).

### 1. Start CockroachDB and create the dev database

```bash
docker run -d --name crdb-amplop \
  -p 26257:26257 -p 8090:8080 \
  cockroachdb/cockroach:latest \
  start-single-node --insecure --store=type=mem,size=1GiB

# wait a moment for it to come up, then create the dev database
docker exec crdb-amplop ./cockroach sql --insecure -e "CREATE DATABASE IF NOT EXISTS devdb;"
```

(`-p 8090:8080` exposes the CockroachDB admin UI at http://localhost:8090; the
in-memory store is wiped on container removal.)

### 2. Apply the migration

```bash
cat migrations/0001_init_amplop.sql | docker exec -i crdb-amplop ./cockroach sql --insecure --database=devdb
```

### 3. Point `.env` at the local node

Copy `.env.example` → `.env` and use the local insecure values:

```bash
DB_HOST=localhost
DB_PORT=26257
DB_USER=root
DB_PASSWORD=
DB_NAME=devdb
DB_SSL_MODE=disable
```

### 4. Run the server

```bash
make run-expense                 # FUNCTION_TARGET=Expense PORT=8080
# or pin time:  make run func=Expense port=8080 time=2026-06-23T10:00:00Z
```

Then hit it:

```bash
curl 'http://localhost:8080/month?year=2026&month=6'
curl -X POST http://localhost:8080/expenses \
  -H 'Content-Type: application/json' \
  -d '{"date":"2026-06-23","time":"12:10","amount":18000,"category":"Makan","note":"Nasi padang"}'
```

### Tests

```bash
go test ./...                              # unit tests (pure engine, services, handlers)

# integration tests run against the local devdb (skipped if DB_HOST is unset):
DB_HOST=localhost DB_PORT=26257 DB_USER=root DB_PASSWORD= DB_NAME=devdb DB_SSL_MODE=disable \
  go test -tags integration ./...
```

## Deploy

Production target is the single `Expense` Cloud Function over the production
CockroachDB (`defaultdb`) with `DB_SSL_MODE=verify-full` (requires the CA cert).
**Apply `migrations/0001_init_amplop.sql` to the production database before the
first deploy.**

```bash
gcloud functions deploy Expense \
  --gen2 \
  --runtime=go121 \
  --region=asia-southeast2 \
  --trigger-http \
  --allow-unauthenticated \
  --entry-point=Expense \
  --set-env-vars=DB_HOST=...,DB_PORT=...,DB_USER=...,DB_PASSWORD=...,DB_NAME=defaultdb,DB_SSL_MODE=verify-full,DB_SSL_ROOT_CERT=ca.crt
```

Notes:
- `--entry-point=Expense` matches the `functions.HTTP("Expense", …)` registration in `function.go`.
- The CA cert (`DB_SSL_ROOT_CERT`) must be deployed alongside the source (it is read at runtime).
- `--allow-unauthenticated` keeps the v2 "no auth, `CORS: *`" model; CORS is handled in-app (`internal/platform/httpx`).
- Do **not** set `DB_SSL_MODE=disable` in production.
