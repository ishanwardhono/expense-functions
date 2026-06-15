# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

Go backend for a personal expense-tracking app, deployed as **Google Cloud Functions** (functions-framework-go) over **CockroachDB**. Single user, no auth, `CORS: *`.

## ⚠️ Active rewrite — read the plan first

The backend is being **completely rewritten** from the v1 model (`weekly/`, `monthly/`, `recap/`, `hello/` — expenses keyed by `year/week/day`, daily allowances, pay-period recap) to the **v2 "Amplop" envelope-budgeting** model. Development spans multiple sessions.

**The authoritative source of truth is the plan:** `docs/superpowers/specs/2026-06-15-amplop-v2-backend-design.md`. Read it before writing v2 code; it distills the Claude Design handoff (the UI prototype) so you don't need the prototype itself. Build in the order of its **§8 phases**, keeping the build green per phase.

**Do not extend the v1 packages.** They are legacy and will be deleted (Phase 0). New work goes under `internal/` per the spec's target layout. `.github/copilot-instructions.md` describes the *old* v1 design and is stale until Phase 4 updates it.

### Locked v2 decisions (see spec §2 for rationale)
- **Server computes** all envelope/budget math; the API returns render-ready data (integers; the client formats).
- New **`amplop` schema** with fresh tables; **no migration** of v1 data. Leave legacy `public.expense` / `public.monthly_expense` untouched.
- **One routed `Expense` Cloud Function** (method+path router) instead of one function per operation.
- **Subscription payments are ordinary expenses** (category `Langganan` + `subscription_id`); single transactions table; **at most one payment per subscription per calendar month**.
- **Budgets + subscriptions are effective-dated**: read the latest version with effective month ≤ viewed month; writes are effective from the **current** month (past months stay frozen).
- AI screenshot import (`/scan`) is **deferred to Phase 2**.

## Commands

```bash
go build ./...                                   # build
go vet ./...                                     # vet
gofmt -l .                                       # list unformatted files
go test ./...                                    # all tests
go test ./internal/envelope/ -run TestName -v    # single package / single test (v2)

# Run one function locally (exports .env, then go run cmd/main.go).
# FUNCTION_TARGET selects which registered function serves; TIME is optional.
make run func=WeeklyGet port=8199 time=2026-06-15T10:00:00Z   # generic form
make run-weekly-get                                            # v1 convenience targets exist for each function
# After the rewrite the single target is `Expense`: make run func=Expense port=8080

docker-compose up weekly-get                     # run a function in Docker (one service per function; v1)
```

There are no tests yet; the v2 plan adds them (pure engine unit tests; repo integration tests against `devdb`). Tests and effective-dated writes rely on the **`TIME`** env var (RFC3339) to pin "now" / the current month — use it for determinism.

## Environment & database

- Copy `.env.example` → `.env` (gitignored). Requires `DB_HOST/PORT/USER/PASSWORD/NAME` and `DB_SSL_ROOT_CERT` (path to the CA cert; `*.crt` is gitignored). v1 also uses `MAX_EXPENSE`/`MAX_MONTHLY_EXPENSE`; v2 moves budgets into the DB.
- CockroachDB connection uses `sslmode=verify-full` (needs the CA cert) and a **single connection** (`SetMaxOpenConns(1)`) — the serverless function pattern. See `common/db.go` (→ `internal/platform/database` in v2).
- **DB targets:** `defaultdb` = production, `devdb` = local/testing.
- Timezone is **Asia/Jakarta** for all date logic (`common/time.go`).

## Architecture (v2 target — the big picture)

Strict layering: **handler (HTTP) → service (orchestration + validation + effective-date resolution) → repo (SQL)**. The budgeting rules live in one **pure** package so they are trivially testable and never duplicated.

```
function.go            registers ONE HTTP function "Expense" → internal router
internal/
  platform/            config · database(sqlx) · httpx(router, CORS, JSON, error→HTTP) · timeutil
  envelope/            PURE engine: EnvelopeOf + ComputeMonth over an already-resolved month context
  expense/             model · repo · service · handler   (all transactions, incl. subscription payments)
  subscription/        identity table + effective-dated versions + resolution
  budget/              effective-dated config + resolution
  month/               resolve config+subs → run engine → assemble the GET /month dashboard
migrations/            0001_init_amplop.sql
```

Key cross-cutting domain rules (full detail in the spec — these require reading several prototype files to grasp, so they are summarized here):

- **Four envelopes, derived (not stored)** from an expense's category + day-of-week: `belanja` (Belanja/Cash any day; Makan/Jajan on weekdays), `weekend` (Makan/Jajan/Lainnya on Sat–Sun), `fleksibel` (Lainnya on weekdays), `langganan` (category Langganan). See `EnvelopeOf` (spec §6.1).
- **Month boundaries:** a shopping week (Mon–Sun) belongs to the month of its **Friday**; a weekend (Sat+Sun) to the month of its **Saturday**. A transaction can show in month *M*'s calendar yet count toward a neighbor month's envelope (spec §6.2). No carry-over between days/weeks.
- **Effective-date resolution** (spec §5.1/§5.2): the engine stays pure by receiving the *resolved* config + subscription set; versioning logic lives in the repo/service.

The `internal/envelope` engine knows nothing about HTTP, the DB, or versioning — keep it that way.

## Conventions

- Errors surface as non-2xx JSON `{"error":"message"}` (current `base.go`; preserved in `internal/platform/httpx`).
- `sqlx` with `db:"col"` struct tags; every DB function takes `context.Context` first; parameterized queries only.
- Money is integer Rupiah (`INT8`). `gen_random_uuid()` primary keys.
- `debug.go` is gitignored (local-only scratch).
