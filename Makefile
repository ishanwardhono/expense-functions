#ex: make run func=HelloGet
run:
	@export $$(cat .env | xargs) && FUNCTION_TARGET=$(func) PORT=$(port) TIME=$(time) go run cmd/main.go

run-weekly-get:
	@make run func=WeeklyGet port=8199

run-weekly-add:
	@make run func=WeeklyAdd port=8198