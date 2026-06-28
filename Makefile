# Run the single routed Expense function.
# ex: make run func=Expense port=8080 time=2026-06-15T10:00:00Z
run:
	@export $$(grep -v '^\s*#' .env | grep -v '^\s*$$' | xargs) && FUNCTION_TARGET=$(func) PORT=$(port) TIME=$(time) go run cmd/main.go

run-expense:
	@make run func=Expense port=8080

# Apply pending DB migrations with the golang-migrate CLI.
# Reads DB_* from .env and builds a cockroachdb:// URL. The sslrootcert query
# param is omitted for sslmode=disable (local insecure node), mirroring the DSN
# logic in internal/platform/database/database.go.
#
# Requires the migrate CLI:
#   go install -tags 'cockroachdb' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
migrate-up:
	@export $$(grep -v '^\s*#' .env | grep -v '^\s*$$' | xargs) && \
	url="cockroachdb://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSL_MODE" && \
	if [ "$$DB_SSL_MODE" != "disable" ]; then url="$$url&sslrootcert=$$DB_SSL_ROOT_CERT"; fi && \
	migrate -path migrations -database "$$url" up

# --- Infrastructure as code (Terraform) --------------------------------------
# Terraform owns the Cloud Run *infrastructure* (service shell, IAM, secret access,
# scaling, CI auth). Code rollouts go through the GitHub Action / `make deploy`.
# See terraform/ and docs/ for the full flow. Requires the `terraform` and `gcloud`
# CLIs and an authenticated gcloud session (`gcloud auth application-default login`).
PROJECT_ID   ?= weekly-expense
REGION       ?= asia-southeast1
SERVICE_NAME ?= expense
TFSTATE_BUCKET ?= weekly-expense-tfstate
GH_REPO      ?= ishanwardhono/expense-functions
DEPLOY_SA    ?= $(SERVICE_NAME)-deployer@$(PROJECT_ID).iam.gserviceaccount.com
INFRA_SA     ?= $(SERVICE_NAME)-infra@$(PROJECT_ID).iam.gserviceaccount.com

# One-time: create the GCS bucket that holds Terraform state (must exist before
# `terraform init` can use the gcs backend).
tf-bootstrap:
	gcloud storage buckets create gs://$(TFSTATE_BUCKET) \
		--project=$(PROJECT_ID) --location=$(REGION) --uniform-bucket-level-access
	gcloud storage buckets update gs://$(TFSTATE_BUCKET) --versioning

# One-time (after the first `tf-apply` creates the infra SA): grant that SA
# read/write on the Terraform state objects. Done out-of-band — not in the module —
# because the infra SA can't manage its own state-bucket IAM from inside its own
# apply (it lacks bucket-level getIamPolicy). Run by an owner with bucket admin.
tf-grant-state:
	gcloud storage buckets add-iam-policy-binding gs://$(TFSTATE_BUCKET) \
		--member="serviceAccount:$(INFRA_SA)" \
		--role="roles/storage.objectAdmin"

tf-init:
	cd terraform && terraform init

tf-plan:
	cd terraform && terraform plan

tf-apply:
	cd terraform && terraform apply

# Push the Terraform outputs straight into the repo's GitHub Actions variables (no
# copy-paste). The deploy workflow reads WIF_PROVIDER + DEPLOY_SA_EMAIL; the terraform
# workflow reads WIF_PROVIDER + TF_INFRA_SA_EMAIL. Requires the `gh` CLI, authenticated
# (`gh auth login`).
gh-vars:
	cd terraform && \
	gh variable set WIF_PROVIDER     --repo $(GH_REPO) --body "$$(terraform output -raw wif_provider)" && \
	gh variable set DEPLOY_SA_EMAIL  --repo $(GH_REPO) --body "$$(terraform output -raw deploy_sa_email)" && \
	gh variable set TF_INFRA_SA_EMAIL --repo $(GH_REPO) --body "$$(terraform output -raw infra_sa_email)"
	@echo "Set WIF_PROVIDER, DEPLOY_SA_EMAIL, TF_INFRA_SA_EMAIL on $(GH_REPO)"

# --- Code deploy -------------------------------------------------------------
# Build the Go image and roll out a new revision. Identical to what the GitHub
# Action runs; use locally to deploy without pushing. Env/secrets/scaling set by
# Terraform are preserved (only the image/revision changes).
deploy:
	gcloud run deploy $(SERVICE_NAME) \
		--source=. --region=$(REGION) \
		--function=Expense --build-service-account=$(DEPLOY_SA) \
		--project=$(PROJECT_ID)

.PHONY: run run-expense migrate-up tf-bootstrap tf-grant-state tf-init tf-plan tf-apply gh-vars deploy
