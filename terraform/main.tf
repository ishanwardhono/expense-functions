# Infrastructure for the v2 routed `Expense` Cloud Run service — the whole module in
# one file. Terraform owns the service *shell* (env, secrets, scaling, identity) plus
# the CI auth plumbing; the deploy pipeline owns the container *image* (see the
# lifecycle block on the service).
#
# All config values live in the `locals` block below — change them there.

terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  # Remote state in GCS (durable, lockable). The bucket is created out-of-band by
  # `make tf-bootstrap` — it must exist before `terraform init`.
  backend "gcs" {
    bucket = "weekly-expense-tfstate"
    prefix = "expense"
  }
}

provider "google" {
  project = local.project_id
  region  = local.region
}

# ---------------------------------------------------------------------------
# Config — edit these
# ---------------------------------------------------------------------------
locals {
  project_id   = "weekly-expense"
  region       = "asia-southeast1"
  service_name = "expense"

  # GCS bucket holding Terraform state (created by `make tf-bootstrap`). Must match
  # the bucket in the `backend "gcs"` block above and the Makefile's TFSTATE_BUCKET.
  tfstate_bucket = "weekly-expense-tfstate"

  # owner/name of the GitHub repo allowed to deploy via Workload Identity Federation.
  github_repo = "ishanwardhono/expense-functions"

  # Existing Secret Manager secrets (created in the console; referenced, not owned).
  db_password_secret = "expense-function-cockroachdb-password" # → env DB_PASSWORD
  db_ca_cert_secret  = "expense-cockroachdb-crt"               # → mounted as a file

  # Where the CA cert is mounted; the cert lands at <mount_path>/<filename>.
  ca_cert_mount_path = "/ca"
  ca_cert_filename   = "expense-cockroachdb-crt"

  # Non-secret DB connection config (DB_SSL_ROOT_CERT is derived below, not here).
  db_env = {
    DB_HOST     = "expense-7822.jxf.gcp-asia-southeast1.cockroachlabs.cloud"
    DB_PORT     = "26257"
    DB_USER     = "expense-function"
    DB_NAME     = "defaultdb"
    DB_SSL_MODE = "verify-full"
  }

  # Image used only for first creation; the deploy pipeline replaces it (and the
  # lifecycle block then ignores it).
  placeholder_image = "us-docker.pkg.dev/cloudrun/container/hello"

  # --- derived ---
  db_ssl_root_cert = "${local.ca_cert_mount_path}/${local.ca_cert_filename}"

  required_apis = [
    "run.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com",
    "secretmanager.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "cloudresourcemanager.googleapis.com",
  ]
}

# ---------------------------------------------------------------------------
# APIs
# ---------------------------------------------------------------------------
# disable_on_destroy = false: tearing down this module must not disable APIs other
# resources in the project may rely on.
resource "google_project_service" "required" {
  for_each = toset(local.required_apis)

  project            = local.project_id
  service            = each.value
  disable_on_destroy = false
}

# ---------------------------------------------------------------------------
# Artifact Registry — image repo + cleanup policy
# ---------------------------------------------------------------------------
# `gcloud run deploy --source` pushes built images to a repo named
# `cloud-run-source-deploy`. Terraform owns it so a cleanup policy keeps only the
# most recent images — otherwise every deploy adds an image that lives forever and
# Artifact Registry storage slowly creeps past the 0.5 GB free tier.
#
# NOTE on an existing project: if this repo already exists (e.g. created by a prior
# `gcloud run deploy`), import it before the first apply, then apply to attach the
# policy:
#   terraform import google_artifact_registry_repository.images \
#     projects/<project>/locations/<region>/repositories/cloud-run-source-deploy
resource "google_artifact_registry_repository" "images" {
  location      = local.region
  repository_id = "cloud-run-source-deploy"
  format        = "DOCKER"
  description   = "Container images built by `gcloud run deploy --source`."

  # Keep at least the 5 most recent images (protects rollback targets even if old)...
  cleanup_policies {
    id     = "keep-recent-5"
    action = "KEEP"
    most_recent_versions {
      keep_count = 5
    }
  }

  # ...and delete anything older than 30 days. KEEP wins over DELETE, so the 5 most
  # recent are never removed even once they age out.
  cleanup_policies {
    id     = "delete-older-than-30d"
    action = "DELETE"
    condition {
      older_than = "2592000s" # 30 days
    }
  }

  depends_on = [google_project_service.required]
}

# ---------------------------------------------------------------------------
# Existing secrets (referenced, not owned)
# ---------------------------------------------------------------------------
# Hold live credentials — read as data sources and granted access, never recreated
# (which would drop the stored versions).
data "google_secret_manager_secret" "db_password" {
  secret_id = local.db_password_secret
}

data "google_secret_manager_secret" "db_ca_cert" {
  secret_id = local.db_ca_cert_secret
}

# ---------------------------------------------------------------------------
# Runtime identity + access
# ---------------------------------------------------------------------------
# Dedicated least-privilege SA (replaces the default compute SA the console-created
# v1 services ran as). It only needs to read the two secrets.
resource "google_service_account" "runtime" {
  account_id   = "${local.service_name}-runtime"
  display_name = "Expense v2 Cloud Run runtime SA"
}

resource "google_secret_manager_secret_iam_member" "runtime_db_password" {
  secret_id = data.google_secret_manager_secret.db_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "runtime_db_ca_cert" {
  secret_id = data.google_secret_manager_secret.db_ca_cert.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime.email}"
}

# ---------------------------------------------------------------------------
# The Cloud Run service
# ---------------------------------------------------------------------------
resource "google_cloud_run_v2_service" "expense" {
  name     = local.service_name
  location = local.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account                  = google_service_account.runtime.email
    max_instance_request_concurrency = 1
    timeout                          = "30s"

    scaling {
      max_instance_count = 1
    }

    containers {
      # Placeholder for first create only; replaced by the deploy pipeline and then
      # ignored (see lifecycle below).
      image = local.placeholder_image

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }

      # Non-secret DB connection config.
      dynamic "env" {
        for_each = local.db_env
        content {
          name  = env.key
          value = env.value
        }
      }

      # Path to the mounted CA cert (kept in sync with the volume mount below).
      env {
        name  = "DB_SSL_ROOT_CERT"
        value = local.db_ssl_root_cert
      }

      # DB password from Secret Manager (never materialized in state as plaintext).
      env {
        name = "DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = data.google_secret_manager_secret.db_password.secret_id
            version = "latest"
          }
        }
      }

      volume_mounts {
        name       = "ca-cert"
        mount_path = local.ca_cert_mount_path
      }
    }

    # CA cert mounted as a file at <mount_path>/<ca_cert_filename>.
    volumes {
      name = "ca-cert"
      secret {
        secret = data.google_secret_manager_secret.db_ca_cert.secret_id
        items {
          version = "latest"
          path    = local.ca_cert_filename
        }
      }
    }
  }

  # The deploy pipeline (gcloud run deploy --source) owns image rollouts. Ignore the
  # image plus the client tags gcloud stamps on each deploy, so `terraform plan`
  # stays clean after a code deploy.
  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
    ]
  }

  depends_on = [google_project_service.required]
}

# Public, unauthenticated invocation — keeps the v2 "single user, no auth, CORS:*"
# model (CORS is handled in-app by internal/platform/httpx).
resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  project  = local.project_id
  location = local.region
  name     = google_cloud_run_v2_service.expense.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# ---------------------------------------------------------------------------
# CI auth: Workload Identity Federation for the deploy workflow
# ---------------------------------------------------------------------------
# Lets GitHub Actions authenticate via short-lived OIDC tokens — no downloaded SA
# JSON key to leak or rotate.
resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github-actions"
  display_name              = "GitHub Actions"
  description               = "OIDC pool for GitHub Actions deploys"

  depends_on = [google_project_service.required]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub OIDC"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
  }

  # Hard-restrict token issuance to this repo. Without a condition, any GitHub repo
  # could mint tokens against the pool.
  attribute_condition = "assertion.repository == \"${local.github_repo}\""

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# Identity the workflow acts as once authenticated.
resource "google_service_account" "deploy" {
  account_id   = "${local.service_name}-deployer"
  display_name = "Expense v2 CI deployer SA"
}

# Roles needed to submit the build (Cloud Build), and — since the deploy SA is also
# the build service account (see deploy.yml's --build-service-account) — to write
# build logs (logging.logWriter), push the image (Artifact Registry), and stage the
# source (GCS), plus roll out a new Cloud Run revision.
resource "google_project_iam_member" "deploy_roles" {
  for_each = toset([
    "roles/run.developer",
    "roles/cloudbuild.builds.editor",
    "roles/artifactregistry.writer",
    "roles/storage.admin",
    "roles/logging.logWriter",
  ])

  project = local.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.deploy.email}"
}

# The deploy SA must be able to deploy the service *as* the runtime SA.
resource "google_service_account_iam_member" "deploy_act_as_runtime" {
  service_account_id = google_service_account.runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deploy.email}"
}

# `gcloud run deploy --source` runs a Cloud Build as the build service account.
# We pass the deploy SA itself (deploy.yml --build-service-account), so it must be
# able to act as itself — Cloud Build requires the submitter to have act-as on the
# build SA. This avoids depending on the project's default compute SA.
resource "google_service_account_iam_member" "deploy_act_as_self" {
  service_account_id = google_service_account.deploy.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deploy.email}"
}

# Allow the GitHub repo's OIDC identities to impersonate the deploy SA.
resource "google_service_account_iam_member" "deploy_wif_binding" {
  service_account_id = google_service_account.deploy.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${local.github_repo}"
}

# ---------------------------------------------------------------------------
# CI infra identity: the `terraform.yml` apply workflow runs as this SA
# ---------------------------------------------------------------------------
# Separate from the deploy SA and deliberately powerful — it manages the entire
# module (service accounts, project IAM, secret IAM, WIF, Cloud Run, APIs, state).
# That broad grant is the cost of auto-applying infra from CI; it is constrained by:
#   - the WIF binding (only THIS repo can impersonate it), and
#   - the apply job's `production` environment approval gate.
# Rotating actual secret VALUES is intentionally not in scope here (values never live
# in git); this SA only manages secret containers/IAM and the rest of the infra.
resource "google_service_account" "infra" {
  account_id   = "${local.service_name}-infra"
  display_name = "Expense v2 CI Terraform (infra) SA"
}

resource "google_project_iam_member" "infra_roles" {
  for_each = toset([
    "roles/run.admin",                       # manage the Cloud Run service + its IAM
    "roles/iam.serviceAccountAdmin",         # create/manage the runtime/deploy/infra SAs
    "roles/iam.workloadIdentityPoolAdmin",   # manage the WIF pool/provider
    "roles/resourcemanager.projectIamAdmin", # set project-level IAM bindings
    "roles/secretmanager.admin",             # manage secret IAM (not values)
    "roles/serviceusage.serviceUsageAdmin",  # enable required APIs
    "roles/artifactregistry.admin",          # manage the image repo + cleanup policy
  ])

  project = local.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.infra.email}"
}

# NOTE: the infra SA's access to the state bucket is granted out-of-band (one-time
# setup: `make tf-grant-state`), NOT here. Managing that bucket-IAM binding from
# inside this module would be a bootstrap chicken-and-egg — reconciling it requires
# storage.buckets.getIamPolicy, which the infra SA (object-level state access only)
# lacks, so the apply would 403 on its own state bucket.

# Apply the Cloud Run service *as* the runtime SA.
resource "google_service_account_iam_member" "infra_act_as_runtime" {
  service_account_id = google_service_account.runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.infra.email}"
}

# Let the GitHub repo's OIDC identities impersonate the infra SA.
resource "google_service_account_iam_member" "infra_wif_binding" {
  service_account_id = google_service_account.infra.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${local.github_repo}"
}

# ---------------------------------------------------------------------------
# Outputs — wire these into GitHub (Actions variables) and use to verify
# ---------------------------------------------------------------------------
output "service_url" {
  description = "Public URL of the Expense Cloud Run service."
  value       = google_cloud_run_v2_service.expense.uri
}

output "wif_provider" {
  description = "Full WIF provider resource name → GitHub repo variable WIF_PROVIDER."
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "deploy_sa_email" {
  description = "CI deployer SA email → GitHub repo variable DEPLOY_SA_EMAIL."
  value       = google_service_account.deploy.email
}

output "infra_sa_email" {
  description = "CI Terraform (infra) SA email → GitHub repo variable TF_INFRA_SA_EMAIL."
  value       = google_service_account.infra.email
}
