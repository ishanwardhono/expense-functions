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
