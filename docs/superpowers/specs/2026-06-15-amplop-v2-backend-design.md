# Amplop v2 Backend — Design & Implementation Plan

**Date:** 2026-06-15
**Status:** Approved design — ready for implementation next session
**Scope:** Complete rewrite of the expense-functions backend to serve the **v2 "Amplop" (envelope budgeting)** design only. The v1 weekly/monthly/recap model is dropped.

---

## 1. Context

The current backend (`weekly/`, `monthly/`, `recap/`, `hello/`) implements a v1 model: expenses keyed by `year/week/day`, daily weekday/weekend allowances, and a pay-period recap. The product has moved on. The **v2 "Amplop"** design (Claude Design handoff, primary file `Kalender Pengeluaran v2 (Amplop).html`) replaces it with an **envelope budgeting** system.

This document is the design + phased plan for rewriting the backend to serve v2. Implementation happens in a later session.

> **Two intentional divergences from the prototype** (user direction, design to be updated later):
> 1. **Subscription payments are ordinary expenses.** The `expense` table is the single source of *all* transactions. You pay a subscription by adding an expense with category `Langganan` linked to the subscription. A subscription can be paid **at most once per calendar month**. The Amplop "Langganan" detail is read-only (informational), not a pay/unpay screen.
> 2. **Budgets and subscriptions are effective-dated (per-month snapshots).** Viewing a past month shows the budgets/subscriptions *as they were*; changes apply to the **current month onward**, never retroactively.

### Source of truth (design handoff)
The prototype lives in the handoff bundle (`~/Downloads/Expense-handoff.zip`). Key modules studied:
- `proto/expense-data.jsx` — categories, date helpers, seed data, month grid.
- `proto/amplop-engine.jsx` — **the envelope math** (`amplopOf`, `computeAmplop`, month-boundary rules). Spec for the server engine.
- `proto/amplop-app.jsx` — app state, actions, what each screen consumes.
- `proto/amplop-components.jsx` — envelope card, calendar, envelope detail sheet, day sheet.
- `proto/expense-components.jsx` — add/edit form, expense row, subscription pay sheet, list/filter.
- `proto/scan-flow.jsx` — AI screenshot import (deferred to Phase 2).

In the prototype all math runs client-side over `localStorage`. The backend becomes the source of truth.

---

## 2. Decisions (locked)

| # | Decision | Choice |
|---|----------|--------|
| D1 | Where does envelope/budget math live? | **Server computes.** Backend owns all rules and returns render-ready data. |
| D2 | AI screenshot import ("Impor dari images")? | **Defer to Phase 2.** Interface documented, not implemented. |
| D3 | Budgets & subscriptions configurability? | **DB-stored & editable.** |
| D4 | Reuse the v1 DB schema / migrate old data? | **No.** Fresh tables in a new `amplop` schema; legacy `public.expense` / `public.monthly_expense` left untouched (drop later). |
| D5 | Deployment shape? | **One HTTP function with an internal REST router.** Fallback: one-function-per-operation if separate URLs are required. |
| D6 | Subscription payments storage? | **As expenses.** Single transactions table; a Langganan expense carries `subscription_id` (the "subcategory"). No payment table, no pay/unpay endpoints. **One payment per subscription per month.** |
| D7 | Per-month history of budgets/subscriptions? | **Effective-dated rows.** A new version is stored only when something changes, stamped with an effective `(year, month)`. Reads use the latest version with effective month ≤ viewed month. Writes are effective from the **current** month. |

### Carried over from the current repo (assumptions)
- Go + GCP Functions Framework (`functions-framework-go`); local dev via `cmd/main.go`.
- CockroachDB via `sqlx` + `lib/pq`. Targets: `defaultdb` = production, `devdb` = local/testing.
- Single user, **no auth**, `CORS: *` (unchanged).
- Timezone **Asia/Jakarta**; `TIME` env var overrides "now" for deterministic tests and to derive the "current month" for effective-dated writes.
- Money is integer Rupiah (`INT8`). Responses return integers; the client formats (`fmtK`/`fmtRp`).

---

## 3. Scope

### In scope (Phase 1–4)
- New `amplop` schema: expenses (incl. subscription payments), subscription identity + effective-dated versions, effective-dated budget config.
- Pure **envelope engine** (server port of `computeAmplop`) operating on already-resolved month context.
- Effective-date **resolution** (repo/service): pick the budget config and subscription set in force for a viewed month.
- REST API: month dashboard (read), expense CRUD, subscription definition CRUD (effective-dated), budget config read/update (effective-dated).
- Layered packages; single routed HTTP function; local run + deploy notes.

### Non-goals
- AI scan/import (Phase 2 — interface only).
- Multi-user / auth.
- v1 data migration.
- Carry-over between days/weeks — **v2 has none** (`carryIn` is always 0; v1 `carryBefore` dropped).
- Future-dating of config changes (writes are effective from the current month only).
- Effective-dating of subscription **name/color** (cosmetic; only `alloc`/`due_day`/`active` are versioned — see §5).

---

## 4. Architecture

### 4.1 Package layout

```
cmd/main.go                  # local dev server (functions-framework StartHostPort)
function.go                  # registers ONE HTTP function: "Expense" → router
internal/
  platform/
    config/      env config (DB, TIME override, tz)
    database/    CockroachDB connection (sqlx), helpers
    httpx/       router, base handler, CORS, JSON encode, error→HTTP mapping
    timeutil/    Asia/Jakarta location, Now(), current-month helper, date helpers
  envelope/
    rules.go         EnvelopeOf(category, date), categories, envelope ids/labels
    engine.go        ComputeMonth over resolved month context (pure)
    engine_test.go   exhaustive table tests incl. month boundaries + Langganan
  expense/
    model.go  repo.go  service.go  handler.go
  subscription/
    model.go  repo.go  service.go  handler.go   # identity + effective-dated versions; resolution
  budget/
    model.go  repo.go  service.go  handler.go   # effective-dated config; resolution
  month/
    service.go  handler.go                       # resolve context → run engine → assemble GET /month
migrations/
    0001_init_amplop.sql
docs/superpowers/specs/2026-06-15-amplop-v2-backend-design.md   # this file
```

**Layering:** `handler` (HTTP) → `service` (orchestration, validation, effective-date resolution) → `repo` (SQL). `envelope` is pure (no DB/HTTP, no knowledge of versioning) — it receives the *resolved* config + subscription set for the month. `month` ties services + engine together.

**Removed:** `weekly/`, `monthly/`, `recap/`, `hello/`, the v1 `handler.go` targets, `data/ddl.sql`. `common/` is refactored into `internal/platform/*`.

### 4.2 Deployment
One Cloud Function target `Expense` exposing a small method+path router; shared middleware: CORS preflight, JSON, panic recovery, error→status mapping (`{"error":"..."}` body, preserving `base.go`). Local dev keeps the Makefile pattern (`FUNCTION_TARGET=Expense PORT=… go run cmd/main.go`); the existing `.env` comment-stripping fix is preserved.

> Fallback (D5): the same `service` layer can be exposed as multiple `functions.HTTP(...)` registrations with no business-logic change.

---

## 5. Data model (CockroachDB, schema `amplop`)

`migrations/0001_init_amplop.sql`:

```sql
CREATE SCHEMA IF NOT EXISTS amplop;

-- Subscriptions: stable identity --------------------------------------------
CREATE TABLE IF NOT EXISTS amplop.subscription (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    name       STRING      NOT NULL,
    color      STRING      NOT NULL DEFAULT '',   -- cosmetic; NOT effective-dated
    created_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT subscription_pk PRIMARY KEY (id)
);

-- Subscriptions: effective-dated attributes ---------------------------------
-- A version applies from (effective_year, effective_month) onward until the
-- next version. "active=false" ends the subscription as of that month.
CREATE TABLE IF NOT EXISTS amplop.subscription_version (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    subscription_id UUID        NOT NULL REFERENCES amplop.subscription (id) ON DELETE CASCADE,
    effective_year  INT2        NOT NULL,
    effective_month INT2        NOT NULL,         -- 1..12
    alloc           INT8        NOT NULL,         -- monthly allocation, > 0
    due_day         INT2        NOT NULL,         -- recurring day-of-month (1..31)
    active          BOOL        NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT subscription_version_pk PRIMARY KEY (id),
    CONSTRAINT subscription_version_alloc_positive CHECK (alloc > 0),
    CONSTRAINT subscription_version_due_day_valid  CHECK (due_day BETWEEN 1 AND 31),
    CONSTRAINT subscription_version_month_valid    CHECK (effective_month BETWEEN 1 AND 12),
    CONSTRAINT subscription_version_uniq UNIQUE (subscription_id, effective_year, effective_month)
);

-- Expenses: single source of ALL transactions (incl. subscription payments) --
CREATE TABLE IF NOT EXISTS amplop.expense (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    occurred_date   DATE        NOT NULL,
    occurred_time   TIME        NULL,             -- "HH:MM"; nullable
    amount          INT8        NOT NULL,         -- Rupiah, > 0
    category        STRING      NOT NULL,         -- Makan|Belanja|Jajan|Cash|Lainnya|Langganan
    subscription_id UUID        NULL REFERENCES amplop.subscription (id),  -- "subcategory"; set iff Langganan
    note            STRING      NOT NULL DEFAULT '',
    -- calendar period of the transaction (drives the once-per-month rule below)
    occurred_year   INT2        AS (EXTRACT(year  FROM occurred_date)::INT2) STORED,
    occurred_month  INT2        AS (EXTRACT(month FROM occurred_date)::INT2) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT expense_pk PRIMARY KEY (id),
    CONSTRAINT expense_amount_positive CHECK (amount > 0),
    CONSTRAINT expense_category_valid
        CHECK (category IN ('Makan','Belanja','Jajan','Cash','Lainnya','Langganan')),
    -- subscription_id present  <=>  category is Langganan
    CONSTRAINT expense_langganan_link
        CHECK ((category = 'Langganan') = (subscription_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS expense_occurred_date_idx   ON amplop.expense (occurred_date);
CREATE INDEX IF NOT EXISTS expense_category_idx        ON amplop.expense (category);
CREATE INDEX IF NOT EXISTS expense_subscription_id_idx ON amplop.expense (subscription_id);

-- A subscription can be paid at most ONCE per calendar month (Langganan rows
-- only; non-subscription expenses have subscription_id NULL and are excluded).
CREATE UNIQUE INDEX IF NOT EXISTS expense_one_sub_payment_per_month
    ON amplop.expense (subscription_id, occurred_year, occurred_month)
    WHERE subscription_id IS NOT NULL;

-- Budget config: effective-dated --------------------------------------------
CREATE TABLE IF NOT EXISTS amplop.budget_config (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    effective_year  INT2        NOT NULL,
    effective_month INT2        NOT NULL,         -- 1..12
    monthly         INT8        NOT NULL,
    shop_weekly     INT8        NOT NULL,
    weekend_budget  INT8        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT current_timestamp(),
    CONSTRAINT budget_config_pk PRIMARY KEY (id),
    CONSTRAINT budget_config_month_valid CHECK (effective_month BETWEEN 1 AND 12),
    CONSTRAINT budget_config_effective_uniq UNIQUE (effective_year, effective_month)
);

-- Baseline so every viewable month resolves (defaults from prototype CFG).
INSERT INTO amplop.budget_config (effective_year, effective_month, monthly, shop_weekly, weekend_budget)
VALUES (2025, 1, 5000000, 600000, 200000)
ON CONFLICT (effective_year, effective_month) DO NOTHING;
```

**Notes**
- `expense.subscription_id` is the "subcategory" link; the CHECK enforces *Langganan ⇔ subscription_id present*. A subscription can be paid **at most once per calendar month** — enforced by the unique partial index `expense_one_sub_payment_per_month` (on the generated `occurred_year`/`occurred_month`) **and** by a service-layer pre-check that returns a friendly error. The client additionally disables already-paid subscriptions on the add-expense screen, but the backend is the authority.
- `occurred_year` / `occurred_month` are STORED generated columns derived from `occurred_date`; they back the once-per-month index and the calendar-month queries (langganan/flex). Editing `occurred_date` re-derives them automatically.
- Subscription **name/color** live on the identity row (cosmetic, not versioned). **alloc/due_day/active** live on `subscription_version` (effective-dated). Ending a subscription = a version with `active=false` from the current month.
- Budget config baseline is seeded at `2025-01` with the prototype defaults (`5,000,000 / 600,000 / 200,000`); pick whatever baseline ≤ the earliest viewable month.
- `occurred_date` + nullable `occurred_time` keep the date-keyed envelope math unambiguous and match the form (date/time edited independently).

### 5.1 Effective-date resolution (the read rule)
For a viewed month `(Y, M)`, the "in force" version is the one with the greatest `effective_year*12 + effective_month` that is `≤ Y*12 + M`.

Budget config:
```sql
SELECT monthly, shop_weekly, weekend_budget
FROM amplop.budget_config
WHERE effective_year*12 + effective_month <= $Y*12 + $M
ORDER BY effective_year*12 + effective_month DESC
LIMIT 1;
```

Subscription set (latest version per subscription, only those active):
```sql
SELECT s.id, s.name, s.color, v.alloc, v.due_day
FROM amplop.subscription s
JOIN LATERAL (
    SELECT alloc, due_day, active
    FROM amplop.subscription_version v
    WHERE v.subscription_id = s.id
      AND v.effective_year*12 + v.effective_month <= $Y*12 + $M
    ORDER BY v.effective_year*12 + v.effective_month DESC
    LIMIT 1
) v ON true
WHERE v.active = true;
```
A subscription created later (its earliest version > M) has no version ≤ M and is correctly absent from earlier months.

### 5.2 Effective-date writes (the change rule)
Writes are stamped with the **current** Asia/Jakarta `(year, month)`:
- **Budget change** — upsert one `budget_config` row for the current month (`ON CONFLICT (effective_year, effective_month) DO UPDATE`).
- **New subscription** — insert identity + one `subscription_version` (current month, `active=true`).
- **Edit alloc/due_day** — upsert `subscription_version` for `(subscription_id, current_year, current_month)`.
- **Remove subscription** — upsert `subscription_version` for the current month with `active=false`.
- **Edit name/color** — update the identity row directly (applies to all months; cosmetic).

Past months keep their older version untouched → frozen. The current and all later months read the new version (until a still-later change supersedes it).

---

## 6. Envelope engine (`internal/envelope`) — pure

Operates on the **already-resolved** month context (config + subscription set passed in). No DB, no versioning logic, "today" injected.

### 6.1 Attribution — `EnvelopeOf(category, date) → belanja | weekend | fleksibel | langganan`
- `Langganan` → **langganan** (any day).
- `Belanja` or `Cash` → **belanja** (any day).
- `Makan` or `Jajan` → weekend day ⇒ **weekend**, else **belanja**.
- `Lainnya` → weekend day ⇒ **weekend**, else **fleksibel**.

"weekend day" = Saturday or Sunday. (This extends `amplopOf` with the Langganan case; subscription payments are now ordinary expenses, so no separate handling.)

### 6.2 Month boundaries (critical — test heavily)
- **Shopping weeks**: one per **Friday** in the viewed month; range Monday..Sunday (`mon = fri−4`, `sun = fri+2`); owned by the month of its Friday.
- **Weekends**: one per **Saturday** in the viewed month; range Sat..Sun (`sun = sat+1`); owned by the month of its Saturday.
- **Flex** and **Langganan**: purely by calendar month (`occurred_date` within the month).

**Consequence to preserve:** a transaction's calendar date can sit in month *M* (shown in *M*'s day list/calendar) while its `belanja`/`weekend` envelope is counted in *M−1*/*M+1* (its week's Friday/Saturday belongs to a neighbor). Day-level "sisa belanja"/"sisa wknd" minis appear only when the day's week/weekend is one of the viewed month's weeks.

### 6.3 `ComputeMonth(...) → MonthResult`
Inputs: expenses (date, amount, category) for the relevant range, **resolved config** (monthly, shopWeekly, weekendBudget), **resolved subscription set** (alloc per sub), year, month, today.

- `weeks[]`: `{ friday, monday, sunday, budget=shopWeekly, spent, left, state }`, `spent` = Σ expenses `EnvelopeOf==belanja` with `monday ≤ date ≤ sunday`.
- `weekends[]`: `{ saturday, sunday, budget=weekendBudget, spent, left, state }`, `spent` = Σ expenses `EnvelopeOf==weekend` with `date ∈ {saturday, sunday}`.
- `subsAlloc` = Σ resolved subscriptions' `alloc`.
- `langgananSpent` = Σ expenses `EnvelopeOf==langganan` in month (i.e. category `Langganan`).
- `flexSpent` = Σ expenses `EnvelopeOf==fleksibel` in month.
- `shopBudget = shopWeekly × len(weeks)`, `shopSpent = Σ weeks.spent`.
- `wkndBudget = weekendBudget × len(weekends)`, `wkndSpent = Σ weekends.spent`.
- `flexBudget = monthly − shopBudget − wkndBudget − subsAlloc` (may be negative).
- `totalSpent = shopSpent + wkndSpent + langgananSpent + flexSpent`.
- `sisa = monthly − totalSpent`.
- `rows[4]`: belanja, weekend, **langganan** `{budget=subsAlloc, spent=langgananSpent}`, fleksibel — each `{ id, label, budget, spent, left, over }`.

`state` (week/weekend pills, port of `weekPillState`): **past** (`sun < today`) → final diff; **current** (`start ≤ today ≤ sun`) → `left`; **future** → `budget`.

### 6.4 Per-subscription status (for the read-only Langganan detail)
For each resolved subscription, derive from this month's single Langganan expense linked to it (at most one — §5): `paid = amount` (absent ⇒ unpaid), `paid_date` = that expense's date, `diff = alloc − paid`, `status ∈ {paid, unpaid}`.

### 6.5 Day helpers
`SpentOf(date)` (Σ all expenses that date), `DayContext(date)` (Friday/weekend/weekday label), `DayMinis(date)` (Terpakai; Sisa belanja if the date's week is in this month; Sisa wknd or Sisa fleksibel).

---

## 7. API contract

One routed function. JSON in/out. Errors: non-2xx with `{"error":"message"}`. `year`/`month` query params default to the current Asia/Jakarta month.

### 7.1 Read — month dashboard
`GET /month?year=YYYY&month=MM` → resolves the effective config + subscription set for the month, runs the engine, and returns one payload:

```jsonc
{
  "period": { "year": 2026, "month": 6, "label": "Juni 2026", "is_current": true },
  "stats":  { "spent": 1234000, "budget": 5000000, "remaining": 3766000 },
  "envelopes": [
    { "id": "belanja",   "label": "Belanja Mingguan", "budget": 2400000, "spent": 980000, "left": 1420000, "over": false },
    { "id": "weekend",   "label": "Akhir Pekan",       "budget": 800000,  "spent": 254000, "left": 546000,  "over": false },
    { "id": "langganan", "label": "Langganan",         "budget": 330000,  "spent": 251000, "left": 79000,   "over": false },
    { "id": "fleksibel", "label": "Fleksibel",         "budget": 1470000, "spent": 8000,   "left": 1462000, "over": false }
  ],
  "belanja_weeks": [
    { "range": "1–7 Jun", "monday": "2026-06-01", "friday": "2026-06-05", "sunday": "2026-06-07",
      "budget": 600000, "spent": 432000, "left": 168000, "state": "past" }
  ],
  "weekends": [
    { "range": "6–7 Jun", "saturday": "2026-06-06", "sunday": "2026-06-07",
      "budget": 200000, "spent": 254000, "left": -54000, "state": "past" }
  ],
  "flex": { "budget": 1470000, "spent": 8000, "left": 1462000 },
  "calendar": [
    { "date": "2026-06-01", "dow": 1, "is_weekend": false, "is_today": false, "spent": 45000 }
  ],
  "days": {
    "2026-06-01": [
      { "id": "…", "date": "2026-06-01", "time": "12:10", "amount": 18000,
        "category": "Makan", "subscription_id": null, "note": "Nasi padang",
        "envelope": { "id": "belanja", "label": "BLNJ" } }
    ],
    "2026-06-05": [
      { "id": "…", "date": "2026-06-05", "time": "09:00", "amount": 186000,
        "category": "Langganan", "subscription_id": "…", "note": "Netflix",
        "envelope": { "id": "langganan", "label": "SUBS" } }
    ]
  },
  "subscriptions": [
    { "id": "…", "name": "Netflix", "color": "#c8403c", "alloc": 187000, "due_day": 5,
      "paid": { "date": "2026-06-05", "amount": 186000 }, "status": "paid" }
  ]
}
```

`days` now includes Langganan expenses as ordinary rows (no injection). `subscriptions[].paid` is derived from this month's Langganan expense linked to each subscription (at most one).

### 7.2 Write — expenses (incl. subscription payments)
- `POST /expenses` `{ date, time?, amount, category, subscription_id?, note? }` → created. `subscription_id` **required iff** `category=="Langganan"` (and must reference an existing subscription); must be null otherwise.
- `PUT /expenses/{id}` `{ date, time?, amount, category, subscription_id?, note? }` → updated (same rule).
- `DELETE /expenses/{id}` → 204.
- A Langganan expense must be **unique per `(subscription_id, calendar month)`**; a conflicting create/update returns **409 Conflict** `{"error":"subscription already paid this month"}`. Updates exclude the row itself; moving the date or subscription re-checks the target month.

### 7.3 Write — subscription definitions (effective-dated)
- `GET /subscriptions?year=&month=` → resolved subscription set for that month (default current).
- `POST /subscriptions` `{ name, color?, alloc, due_day }` → creates identity + version effective from current month.
- `PUT /subscriptions/{id}` `{ name?, color?, alloc?, due_day? }` → name/color update the identity; alloc/due_day upsert a version effective from the current month.
- `DELETE /subscriptions/{id}` → upsert a version with `active=false` effective from the current month (soft end; past months keep showing it).

> No payment endpoints. Paying/un-paying a subscription = adding/deleting a `Langganan` expense (§7.2).

### 7.4 Budget config (effective-dated)
- `GET /budget?year=&month=` → resolved config for that month (default current).
- `PUT /budget` `{ monthly, shop_weekly, weekend_budget }` → upsert a version effective from the current month.

### 7.5 Validation (service layer)
- `amount` integer > 0; `category` ∈ allowed set; `date` `YYYY-MM-DD`; `time` `HH:MM` or omitted.
- `category=="Langganan"` ⇒ `subscription_id` present + references an existing subscription; otherwise `subscription_id` must be null.
- `category=="Langganan"` ⇒ at most one such expense per `(subscription_id, calendar month)` — service pre-check (excluding the current row on update) **plus** graceful handling of the DB unique-index violation as a backstop; conflict ⇒ **409**.
- Subscription `alloc` > 0; `due_day` 1–31; `name` non-empty.
- Budget values ≥ 0.

---

## 8. Implementation phases (next-session plan)

Each phase ends green (compiles, tests pass). TDD for the engine and the effective-date resolution.

### Phase 0 — Scaffold
- [ ] `internal/platform/{config,database,httpx,timeutil}` (port from `common/`); add current-month helper honoring `TIME`.
- [ ] `migrations/0001_init_amplop.sql` (all tables + the once-per-month unique partial index in §5); document applying to `devdb` (local) and `defaultdb` (prod).
- [ ] `function.go` registering `Expense` + router with CORS/JSON/error middleware (ported from `base.go`).
- [ ] Update `cmd/main.go`/Makefile/docker-compose to the single `Expense` target.
- [ ] Delete v1 packages + `data/ddl.sql`. Build stays green.

### Phase 1 — Envelope engine (TDD, pure)
- [ ] `rules.go`: categories (incl. `Langganan`), envelope ids/labels, `EnvelopeOf`.
- [ ] `engine.go`: `dowsInMonth`, week/weekend construction, `ComputeMonth` (over resolved context), day helpers, per-sub status helper.
- [ ] `engine_test.go`: June-2026 seed reproduction + boundary cases (week owned by Friday across month edges; weekend by Saturday; flex/langganan by calendar month; transaction visible in *M* but counted in a neighbor's envelope; negative `left`/`flexBudget`; empty month; Langganan attribution).

### Phase 2 — Persistence + effective-date resolution
- [ ] Repos: `expense` (incl. `subscription_id`, month-range + per-sub Langganan queries), `subscription` + `subscription_version`, `budget_config`.
- [ ] Resolution queries (§5.1) for config and subscription set; upsert writes (§5.2).
- [ ] Integration tests against local `devdb`, including: effective-dating (a change in month X is frozen for X−1 and applies X→forward; subscription created later is absent from earlier months; soft-end keeps past months) **and** the once-per-month unique partial index rejecting a duplicate Langganan payment.

### Phase 3 — Services + HTTP
- [ ] Services with validation: expenses (Langganan/subscription_id rule **and the once-per-month payment rule** — service pre-check + graceful unique-violation handling → 409), subscriptions, budget; writing effective-from-current-month versions.
- [ ] `month` service: resolve config + subs → run engine → assemble §7.1 payload (incl. derived `subscriptions[].paid`).
- [ ] Handlers + route wiring for all §7 endpoints; handler tests (happy + validation, incl. duplicate-payment 409) with `TIME` pinned.

### Phase 4 — Run & deploy
- [ ] Verify local run (Makefile target, docker-compose service).
- [ ] Deploy notes for the single `Expense` Cloud Function; env vars; CORS.
- [ ] Update `README` / copilot-instructions to the new structure.

### Phase 5 — AI scan import (deferred; see §9)

---

## 9. Phase 2 (deferred) — AI screenshot import
Interface sketch only.
- `POST /scan` (multipart images) → multimodal **Claude** extracts candidate transactions `{ name, amount, date, time, category, direction: out|in, confidence }`; income flagged/excluded by default; low-confidence marked.
- Client reviews/edits, commits kept `out` items via `POST /expenses:batch` `{ items: [ { date, time?, amount, category, subscription_id?, note? } ] }`. The batch commit must also honor the once-per-month Langganan rule (reject or dedupe conflicting payments).
- Open: model + image-size limits, generic vision vs per-app parsers, whether to store source images (likely discard).

---

## 10. Testing strategy
- **Engine:** pure unit tests, deterministic via injected "today"; mirror prototype outputs for the June-2026 seed; cover Langganan + boundaries.
- **Resolution + repos:** integration tests on `devdb` (CockroachDB) for effective-dating semantics.
- **Once-per-month payment:** unit-test the service pre-check (incl. update excluding self, and moving date/subscription into an occupied month) and an integration test asserting the DB unique partial index rejects a duplicate → surfaced as 409.
- **Services/handlers:** validation + error mapping; `TIME` pins "now" (and the current month for writes).
- CI green per phase.

---

## 11. Open items (with defaults)
1. **Legacy tables**: leave `public.expense` / `public.monthly_expense` untouched; drop after v2 is live. *(Default: leave.)*
2. **Endpoint shape**: single rich `GET /month` + granular writes. *(Default: yes.)*
3. **Deploy granularity**: single routed `Expense` function. *(Default: D5.)*
4. **Effective-from for writes**: current month only (no future-dating). *(Default; revisit if planning ahead is wanted.)*
5. **Name/color versioning**: not versioned (cosmetic). *(Default; move to `subscription_version` if historical names matter.)*
6. **One Langganan payment per sub per month**: enforced (unique partial index + service pre-check → 409). Client also disables already-paid subscriptions on the add-expense screen; backend is the authority. *(Locked per user.)*
7. **Budget baseline month**: seeded at `2025-01`. *(Adjust to the earliest month you'll ever view.)*
8. **Paying a subscription in a month where it isn't effective/active**: not blocked — only the subscription identity must exist. *(Default: allow; add a check if you want payments restricted to active months.)*

---

## 12. Design-file → backend map (reference)
| Prototype | Backend home |
|-----------|--------------|
| `amplop-engine.jsx` (`amplopOf`, `computeAmplop`, boundaries) | `internal/envelope` (extended with `Langganan`) |
| `expense-data.jsx` (`CATS`, date helpers, seed) | `envelope` rules + repo seeds/tests |
| `amplop-app.jsx` (save/delete expense, navMonth) | `expense` service + `GET /month` |
| `amplop-components.jsx` (EnvelopeCard, Calendar, EnvelopeSheet, DaySheet) | `GET /month` payload (§7.1); Langganan detail is read-only (§6.4) |
| `expense-components.jsx` `ExpenseForm` | `POST/PUT /expenses` incl. `subscription_id` for Langganan (once-per-month) |
| `expense-components.jsx` `PaySheet` | **replaced** by add-expense (category Langganan, once per month); no payment endpoints |
| `expense-components.jsx` `SubsSection`/`ListSection` | subscription definition CRUD (§7.3) + `GET /month` |
| `scan-flow.jsx` (ScanFlow, SCAN_RESULT) | Phase 2 `POST /scan` + `POST /expenses:batch` (§9) |
```
