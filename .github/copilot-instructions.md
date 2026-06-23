# GitHub Copilot Custom Instructions for the Expense Function

## Project Overview
Go backend for a personal expense-tracking app, deployed as a single **Google
Cloud Function** (functions-framework-go) over **CockroachDB**. Single user, no
auth, `CORS: *`. This is the **v2 "Amplop" (envelope budgeting)** model.

The authoritative design is
`docs/superpowers/specs/2026-06-15-amplop-v2-backend-design.md`; project
conventions live in `/CLAUDE.md`. Read those before non-trivial changes.

## Architecture & Tech Stack
- **Language**: Go (module targets 1.21)
- **Framework**: Google Cloud Functions Framework (`functions-framework-go`)
- **Database**: CockroachDB via `sqlx` + `lib/pq` (`sslmode=verify-full` in prod;
  `disable` for a local insecure node via `DB_SSL_MODE`). Single connection
  (`SetMaxOpenConns(1)`) — the serverless pattern.
- **ID Generation**: `gen_random_uuid()` / `google/uuid`
- **Money**: integer Rupiah (`INT8`); the API returns integers, the client formats.
- **Timezone**: Asia/Jakarta; `TIME` env var (RFC3339) pins "now" / current month.

## Project Structure
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

## Code Conventions & Patterns

### Layering
- **handler (HTTP) → service (orchestration, validation, effective-date
  resolution) → repo (SQL)**. Keep handlers thin.
- `internal/envelope` is **pure**: no DB, no HTTP, no versioning logic. It
  receives the already-resolved config + subscription set for a month. Keep it that way.

### Database operations
- `sqlx` with `db:"col"` struct tags; every DB function takes `context.Context` first.
- Parameterized queries only. Effective-date reads use a tuple comparison
  `(effective_year, effective_month) <= ($Y, $M)` (range scan; no `year*12+month`).
- New tables live in the `amplop` schema. Do not touch legacy `public.*` tables.

### HTTP & errors
- One routed function registered as `functions.HTTP("Expense", …)` in `function.go`.
- Services return **typed errors** from `internal/platform/apierr`; `httpx` maps
  them to status: `Invalid`→400, `NotFound`→404, `Conflict`→409, else 500. Body is
  always `{"error":"message"}`.
- CORS (`*`), JSON, and panic recovery are middleware in `internal/platform/httpx`.

### Dates / money
- Stored as SQL `DATE` + `TIME` (Asia/Jakarta); the API returns `occurred_at` as
  **RFC3339**. Money is integer Rupiah.

### Testing
- Pure engine + service/handler unit tests run under `go test ./...`.
- Repo integration tests are tagged `//go:build integration` and run against a
  local `devdb` (skipped when `DB_HOST` is unset). See `README.md` for the
  Docker CockroachDB setup.

## Key Domain Rules (see spec §6 for detail)
- **Four envelopes, derived (not stored)** from category + day-of-week: `belanja`,
  `weekend`, `fleksibel`, `langganan` (`EnvelopeOf`).
- **Subscription payments are ordinary expenses** (category `Langganan` +
  `subscription_id`); **at most one payment per subscription per calendar month**
  (unique partial index + service pre-check → 409).
- **Budgets and subscriptions are effective-dated**: reads pick the latest version
  with effective month ≤ viewed month; writes are effective from the current month
  (past months stay frozen).
- **Month boundaries**: a shopping week (Mon–Sun) belongs to the month of its
  Friday; a weekend (Sat+Sun) to the month of its Saturday. The month read query
  loads a window wider than the calendar month; the engine attributes precisely.

## Development Guidelines
- One logical change per PR; focused diffs. Conventional commits.
- Cover both success and failure paths in tests (400/404/409 mappings; effective-dating).
- Don't extend the deleted v1 model; build per the spec's phases, keeping the build green.
