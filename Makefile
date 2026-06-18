# Run the single routed Expense function.
# ex: make run func=Expense port=8080 time=2026-06-15T10:00:00Z
run:
	@export $$(grep -v '^\s*#' .env | grep -v '^\s*$$' | xargs) && FUNCTION_TARGET=$(func) PORT=$(port) TIME=$(time) go run cmd/main.go

run-expense:
	@make run func=Expense port=8080
