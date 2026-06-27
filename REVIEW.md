# Review — PR #15 (Terraform IaC + CI/CD for the v2 Expense Cloud Run service)

Infrastructure-only PR. No Go code changed. Reviewed by hand (Terraform not
installed; `terraform validate`/`plan` not run).

## Blocking

_None._

## Non-blocking (nit)

- [ ] WIF binding scope: both `deploy_wif_binding` and `infra_wif_binding`
  (terraform/main.tf:288-296, 340-346) grant `roles/iam.workloadIdentityUser`
  to `principalSet://.../attribute.repository/<repo>` — i.e. ANY workflow/branch
  in the repo can impersonate either SA, including the powerful `expense-infra`
  SA. The `terraform.yml` *plan* job runs on `pull_request` and authenticates as
  the infra SA with NO `environment:` gate. Acceptable under the documented
  single-user / owner-only model (CLAUDE.md), and the *apply* path is gated by
  the `production` environment. If the repo is ever made public or gains outside
  collaborators, tighten the WIF `attribute_condition` / principalSet to a
  branch (`attribute.ref == "refs/heads/main"`) or to a dedicated environment,
  and/or split a read-only plan SA from the apply SA.

- [ ] `gcloud run deploy --source=.` (deploy.yml / `make deploy`) submits a Cloud
  Build. The deploy SA has `cloudbuild.builds.editor` + `artifactregistry.writer`
  + `storage.admin`, but the buildpacks build itself runs under the Cloud Build
  service identity; depending on gcloud/project setup the first build may also
  need `roles/cloudbuild.builds.builder` (or the project's Cloud Build default
  SA). Surfaces only at first deploy, which the PR explicitly defers to manual
  validation — flagging for awareness.

## Notes (verified, no action)

- DB_SSL_ROOT_CERT derivation: cert mounted at `/ca` with filename
  `expense-cockroachdb-crt`; `db_ssl_root_cert = "/ca/expense-cockroachdb-crt"`
  set as env. Matches `config.LoadDatabase` reading `DB_SSL_ROOT_CERT` as the
  cert file path. Correct. (main.tf:53-67, 167-172, 184-200)
- Resource references sound:
  - secret IAM uses data-source `.id` (full path) — valid for
    `google_secret_manager_secret_iam_member.secret_id`.
  - Cloud Run v2 `secret`/`secret_key_ref.secret` use `.secret_id` (short name,
    same project) — valid.
  - WIF principalSet/output use `pool.name` / `provider.name` (full resource
    names) — correct formats.
  - `service_account_iam_member.service_account_id` uses SA `.name`
    (`projects/.../serviceAccounts/<email>`) — correct.
- `lifecycle.ignore_changes = [template[0].containers[0].image, client,
  client_version]` is the correct addressing for this resource; lets deploy.yml
  own the image while Terraform owns the rest. Placeholder image used only on
  first create. (main.tf:204-214)
- No secret VALUES committed — secrets are referenced via data sources only.
  `terraform/.gitignore` excludes `*.tfstate*`, `*.tfplan`, `.terraform/`,
  override files; keeps `.terraform.lock.hcl`.
- CI variable wiring is consistent: `make gh-vars` sets WIF_PROVIDER /
  DEPLOY_SA_EMAIL / TF_INFRA_SA_EMAIL from TF outputs; deploy.yml reads
  WIF_PROVIDER + DEPLOY_SA_EMAIL; terraform.yml reads WIF_PROVIDER +
  TF_INFRA_SA_EMAIL. All `id-token: write` + `contents: read` present; apply +
  deploy gated by `environment: production`.
- Pre-existing `migrations-deploy.yml` uses GitHub `secrets.DB_*` (not WIF) — no
  conflict with the new SAs.
- `allUsers` invoker is intentional per CLAUDE.md (single user, no auth, CORS:*)
  — not a finding.
- Makefile / workflows / README consistent: region `asia-southeast1`, project
  `weekly-expense`, service `expense`, `--function=Expense` matches
  `functions.HTTP("Expense", …)`, defaultdb + verify-full.
- Chicken-and-egg: first `tf-apply` runs locally under user ADC (not WIF), so
  the infra SA + WIF login need not pre-exist. Sound.
