#ex: make run func=HelloGet
run:
	@export $$(cat .env | xargs) && FUNCTION_TARGET=$(func) PORT=$(port) TIME=$(time) go run cmd/main.go

run-weekly-get:
	@make run func=WeeklyGet port=8199

run-weekly-add:
	@make run func=WeeklyAdd port=8198

run-monthly-get:
	@make run func=MonthlyGet port=8197

run-monthly-add:
	@make run func=MonthlyAdd port=8196

run-hello:
	@make run func=Hello port=8100