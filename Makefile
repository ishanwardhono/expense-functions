#ex: make run func=HelloGet
run:
	@export $$(cat .env | xargs) && FUNCTION_TARGET=$(func) TIME=$(time) go run cmd/main.go
