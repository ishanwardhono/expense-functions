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

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate)
(versioned, up-only `*.up.sql` files under `migrations/`, tracked in a
`schema_migrations` ledger). Install the CLI once:

```bash
go install -tags 'cockroachdb' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
```

Then apply pending migrations (reads `DB_*` from `.env`, so do step 3 first or
export them):

```bash
make migrate-up
```

> Re-running is safe — already-applied migrations are skipped via the ledger.
> The very first `make migrate-up` against a DB that was previously set up by
> hand simply records the baseline (migration `0001` is idempotent).

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

## Infrastructure & deployment

Everything is code — no console clicks. There are **three things you ever do**, and
each maps to a clear command or a git push:

| # | You want to… | How |
|---|---|---|
| 1 | **Create the function** on a (new) GCP account | run the one-time setup commands below |
| 2 | **Change infra** (an env var, scaling, a secret reference) | edit `terraform/main.tf`, open a PR, merge → auto-applies |
| 3 | **Ship new code** | push Go changes to `main` → auto-deploys |

### How it fits together

- **`terraform/main.tf`** is the infrastructure (what the console used to hold): the
  single `expense` Cloud Run service (region `asia-southeast1`, over production
  `defaultdb`, `DB_SSL_MODE=verify-full`), its runtime service account, Secret Manager
  access, and the CI login (Workload Identity Federation). All editable config sits in
  the `locals` block at the top of the file. State lives in a GCS bucket.
- **Two GitHub workflows** do the automation, each on its own trigger:
  - **`deploy.yml`** — on push to `main` touching `*.go` → builds the image and rolls
    out a new Cloud Run revision (your **need #3**).
  - **`terraform.yml`** — on a PR touching `terraform/**` it posts a **plan**; on merge
    to `main` it **applies** (your **need #2**). Both behind the `production` approval gate.
- The two never fight over the container image: a `lifecycle { ignore_changes = [image] }`
  on the service lets `deploy.yml` own the image while Terraform owns everything else.

### Identities (who can do what)

Terraform creates three service accounts, least-privilege:

| SA | Used by | Can |
|---|---|---|
| `expense-runtime` | the running service | read the 2 DB secrets |
| `expense-deployer` | `deploy.yml` | build + deploy a revision |
| `expense-infra` | `terraform.yml` apply | manage the whole module (IAM, SAs, secrets, WIF) — powerful; impersonation is locked to this repo and apply is gated |

CI authenticates via **Workload Identity Federation** (short-lived OIDC tokens) — no
service-account JSON keys are ever created or stored.

### Need #1 — first-time setup, start to finish (run once)

Everything before "Done" below is one-time. The first `terraform apply` runs **locally**
(the infra SA + its CI login don't exist yet); after that, CI takes over.

**Step 0 — install the tools** (once per machine):

```bash
# Terraform lives in HashiCorp's tap, not Homebrew core:
brew tap hashicorp/tap
brew install hashicorp/tap/terraform
brew install gh                # gcloud assumed already installed
```

**Step 1 — log in locally:**

```bash
gcloud auth login                          # your human login
gcloud auth application-default login      # the credential Terraform actually uses
gcloud config set project weekly-expense
gh auth login                              # so `make gh-vars` can set GitHub variables
```

**Step 2 — confirm the two DB secrets exist** (Terraform references, doesn't create them):

```bash
gcloud secrets list --filter="name~expense"
# expect: expense-function-cockroachdb-password  AND  expense-cockroachdb-crt
```

On a brand-new project, create them first (`gcloud secrets create … && gcloud secrets
versions add …` for the password, and the CA cert).

**Step 3 — create the infrastructure** (from your laptop, on the repo checkout):

```bash
make tf-bootstrap   # create the GCS bucket that stores Terraform state
make tf-init        # download the Google provider
make tf-apply       # review the plan, type "yes" — creates the service + 3 SAs + WIF
make tf-grant-state # let the infra SA read/write state (so CI applies can run)
make gh-vars        # push WIF_PROVIDER, DEPLOY_SA_EMAIL, TF_INFRA_SA_EMAIL into GitHub
```

`make tf-grant-state` grants the `expense-infra` SA `roles/storage.objectAdmin` on
the state bucket. It is done here (out-of-band, by you) rather than in the module:
the infra SA can't manage its own state-bucket IAM from inside its own apply, so
Terraform owning that binding would 403 every CI apply.

`make gh-vars` sets the repo **Actions variables** the workflows read (via `gh`, no
copy-paste). Until they exist, both workflows **skip** (they guard on
`vars.WIF_PROVIDER`), so they don't fail before the infra exists.

**Step 4 — apply DB migrations to production** (if not already done): the app needs the
`amplop` schema before it can serve traffic (see
[Database migrations in CI](#database-migrations-in-ci)).

**Step 5 — first code deploy** (`tf-apply` left a placeholder image; replace it once):

```bash
make deploy         # builds the Go code and rolls out the real revision
```

**Step 6 — verify:**

```bash
cd terraform && terraform output -raw service_url      # the public URL
curl 'https://<that-url>/month?year=2026&month=6'      # expect JSON, not the hello page
```

**Step 7 — merge the infra PR** so both workflows live on `main` and the automation is active.

**Step 8 (optional, recommended) — turn on the approval gates:** GitHub → Settings →
Environments → **New environment** → `production` → add yourself as a required reviewer.
Without it, deploys/applies just run automatically (no pause).

**Done.** From here on, use Need #2 and Need #3 below.

### Decommissioning the v1 functions (one-time)

The v2 backend is a **single** `expense` Cloud Run service (`--function=Expense`).
If the project still holds the old v1 functions (`WeeklyGet`, `MonthlyGet`,
`RecapGet`, `hello`, …) from before the rewrite, delete them so only `expense`
remains. These are not Terraform-managed — remove them out-of-band with `gcloud`:

```bash
# Discover what's still deployed
gcloud functions list --project=weekly-expense
gcloud run services list --project=weekly-expense --region=asia-southeast1

# Delete each stale v1 function (keep `expense`):
gcloud functions delete <NAME> --region=asia-southeast1 --project=weekly-expense
# …or, if it shows up as a Cloud Run service instead:
gcloud run services delete <NAME> --region=asia-southeast1 --project=weekly-expense
```

### Need #2 — change infrastructure

Edit the value in the `locals` block of `terraform/main.tf` (e.g. add an env var, bump
`max_instance_count`, point at a different secret), then:

```bash
make tf-plan        # optional: preview the change locally
# commit on a branch, open a PR
```

On the PR, the **terraform** workflow posts the plan. **Merge to `main`** → it applies
(after the `production` approval click). Nothing else to run.

> Secret **values** are never in git. Terraform manages the secret *references*; rotate
> an actual password/cert with `gcloud secrets versions add …` out-of-band.

### Need #3 — ship new code

Just push Go changes to `main` (paths under `*.go`/`go.mod`/`go.sum`). The **deploy**
workflow builds and rolls out a new revision automatically (gated by `production`).
Env vars, secrets, scaling, and the runtime SA set by Terraform are preserved — only
the image changes.

> Need to deploy from your laptop without pushing? `make deploy` runs the exact same
> `gcloud run deploy` the workflow uses.

Notes:
- `--function=Expense` matches the `functions.HTTP("Expense", …)` registration in `function.go`.
- The CA cert and DB password are mounted from Secret Manager by Terraform (not passed on the deploy command).
- The service is public/unauthenticated — keeps the v2 "no auth, `CORS: *`" model; CORS is handled in-app (`internal/platform/httpx`).
- Do **not** set `DB_SSL_MODE=disable` in production.

### Cost

For a single user with `max_instance_count = 1`, this runs comfortably inside the GCP
**Always Free** tier (2M Cloud Run requests, 180k vCPU-sec, 360k GiB-sec, 2,500 Cloud
Build minutes/month) — effectively **$0/month**. Linking a billing account moves the
Firebase project to the **Blaze** plan, which is expected: Blaze just means "billing
attached" and still includes all the free quotas.

- The one cost that quietly grows is **Artifact Registry storage** (0.5 GB free) — each
  deploy adds an image. Terraform owns the `cloud-run-source-deploy` repo with a
  **cleanup policy** (keep the 5 most recent, delete images older than 30 days) so it
  never creeps past the free tier.
- **On an existing project** the repo likely already exists (from a prior deploy), so
  import it before the first apply:
  ```bash
  cd terraform && terraform import google_artifact_registry_repository.images \
    projects/weekly-expense/locations/asia-southeast1/repositories/cloud-run-source-deploy
  ```
- Recommended safety net: set a **budget alert** (Console → Billing → Budgets & alerts),
  e.g. $5/month with alerts at 50/90/100%. It emails you; it does not cap spend.

## Database migrations in CI

Two GitHub Actions workflows manage migrations:

- **`migrations-check`** (`.github/workflows/migrations-check.yml`) — runs on every
  PR that touches `migrations/**`. Spins up a throwaway CockroachDB node and
  applies all migrations from a clean slate, so a malformed or out-of-order
  migration fails review instead of prod. No secrets required.
- **`migrations-deploy`** (`.github/workflows/migrations-deploy.yml`) — applies
  pending migrations to production (`defaultdb`, `verify-full`) on merge to
  `main`.

### Prod auto-apply ships DISABLED

The deploy job is gated by the repo variable **`MIGRATIONS_AUTO_APPLY`**:

- Unset / not `true` → merges to `main` trigger the workflow but the apply job is
  **skipped**. Nothing touches prod. (This is the current state — still testing.)
- Set to `true` (GitHub → Settings → Secrets and variables → Actions →
  **Variables**) → merges auto-apply.
- A manual **Run workflow** (`workflow_dispatch`) runs regardless of the flag —
  use it to validate the cert/secrets path while testing. Its optional
  `database` input overrides the target DB name.

Every prod run also pauses on the `production` **Environment** approval gate
(GitHub → Settings → Environments → `production` → required reviewers) before any
DDL executes.

### Required secrets

Set these as repo or `production`-environment secrets for `migrations-deploy`:

| Secret | Value |
| --- | --- |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` | Production CockroachDB connection |
| `DB_NAME` | `defaultdb` |
| `DB_SSL_ROOT_CERT_B64` | The CA cert, base64-encoded: `base64 -i ca.crt` (the workflow decodes it to a temp file) |

### Local migrations

```bash
make migrate-up      # applies pending migrations using DB_* from .env
```
